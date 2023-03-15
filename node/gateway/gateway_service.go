package gateway

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sao-node/chain"
	"sao-node/node/config"
	"sao-node/store"
	"sao-node/types"
	"sao-node/utils"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
)

var log = logging.Logger("gateway")

const (
	WINDOW_SIZE       = 2
	SCHEDULE_INTERVAL = 1
	LOCKNAME_COMPLETE = "complete"
)

type CommitResult struct {
	OrderId uint64
	DataId  string
	Commit  string
	Commits []string
	Cid     string
	Shards  map[string]*saotypes.ShardMeta
}

type FetchResult struct {
	Cid     string
	Content []byte
}

type GatewaySvcApi interface {
	QueryMeta(ctx context.Context, req *types.MetadataProposal, height int64) (*types.Model, error)
	CommitModel(ctx context.Context, clientProposal *types.OrderStoreProposal, orderId uint64, content []byte) (*CommitResult, error)
	FetchContent(ctx context.Context, req *types.MetadataProposal, meta *types.Model) (*FetchResult, error)
	TerminateOrder(ctx context.Context, req *types.OrderTerminateProposal) error
	RenewOrder(ctx context.Context, req *types.OrderRenewProposal) (map[string]string, error)
	UpdateModelPermission(ctx context.Context, req *types.PermissionProposal) error
	Stop(ctx context.Context) error
	OrderStatus(ctx context.Context, id string) (types.OrderInfo, error)
	OrderFix(ctx context.Context, id string) error
	OrderList(ctx context.Context) ([]types.OrderInfo, error)
}

type WorkRequest struct {
	Order types.OrderInfo
}

type GatewaySvc struct {
	ctx                context.Context
	chainSvc           *chain.ChainSvc
	storeManager       *store.StoreManager
	keyringHome        string
	nodeAddress        string
	localPeerId        string
	stagingPath        string
	cfg                *config.Node
	orderDs            datastore.Batching
	gatewayProtocolMap map[string]GatewayProtocol

	schedule   chan *WorkRequest
	schedQueue *RequestQueue
	locks      *utils.Maplock

	completeResultChan chan string
	completeMap        map[string]int64
}

func NewGatewaySvc(
	ctx context.Context,
	nodeAddress string,
	chainSvc *chain.ChainSvc,
	host host.Host,
	cfg *config.Node,
	storeManager *store.StoreManager,
	notifyChan map[string]chan interface{},
	orderDs datastore.Batching,
	keyringHome string,
) *GatewaySvc {
	cs := &GatewaySvc{
		ctx:                ctx,
		chainSvc:           chainSvc,
		storeManager:       storeManager,
		keyringHome:        keyringHome,
		nodeAddress:        nodeAddress,
		localPeerId:        host.ID().String(),
		stagingPath:        cfg.Transport.StagingPath,
		cfg:                cfg,
		completeResultChan: make(chan string),
		completeMap:        make(map[string]int64),
		orderDs:            orderDs,
		schedule:           make(chan *WorkRequest),
		schedQueue:         &RequestQueue{},
		locks:              utils.NewMapLock(),
	}
	cs.gatewayProtocolMap = make(map[string]GatewayProtocol)

	local := NewLocalGatewayProtocol(
		ctx,
		notifyChan,
		storeManager,
		cs,
	)
	cs.gatewayProtocolMap["local"] = local
	cs.gatewayProtocolMap["stream"] = NewStreamGatewayProtocol(
		ctx,
		host,
		cs,
		local,
	)

	go cs.runSched(ctx, host)
	go cs.processIncompleteOrders(ctx)
	go cs.completeLoop(ctx)

	return cs
}

func (gs *GatewaySvc) completeLoop(ctx context.Context) {
	for {
		select {
		case dataId := <-gs.completeResultChan:
			gs.locks.Lock("complete")
			delete(gs.completeMap, dataId)
			gs.locks.Unlock("complete")
		case <-time.After(time.Minute):
		case <-ctx.Done():
			return
		}
		// TODO: handle wait completion timeout.
	}
}

func (gs *GatewaySvc) processIncompleteOrders(ctx context.Context) {
	log.Info("process pending orders...")
	pendings, err := gs.getPendingOrders(ctx)
	if err != nil {
		log.Error("process pending orders error: %v", err)
	} else {
		for _, p := range pendings {
			gs.schedule <- &WorkRequest{
				Order: p,
			}
		}
	}
}

func (gs *GatewaySvc) runSched(ctx context.Context, host host.Host) {
	for {
		select {
		case req := <-gs.schedule:
			gs.schedQueue.Push(req)
		case <-time.After(time.Minute * SCHEDULE_INTERVAL):
		}

		throttle := make(chan struct{}, WINDOW_SIZE)

		var reschedule []types.OrderInfo
		len := gs.schedQueue.Len()
		var wg sync.WaitGroup
		wg.Add(len)
		for i := 0; i < len; i++ {
			throttle <- struct{}{}

			go func(sqi int) {
				defer wg.Done()
				defer func() {
					<-throttle
				}()

				task := (*gs.schedQueue)[sqi]
				err := gs.process(ctx, task.Order)
				if err != nil {
					log.Warnf("process order %d error: %v", task.Order.OrderId, err)
					newOrder, err := utils.GetOrder(ctx, gs.orderDs, task.Order.DataId)
					if err != nil {
						reschedule = append(reschedule, newOrder)
					} else {
						reschedule = append(reschedule, task.Order)
					}
				}
			}(i)
		}
		wg.Wait()

		for i := 0; i < len; i++ {
			gs.schedQueue.Remove(0)
		}
		for _, r := range reschedule {
			gs.schedQueue.Push(&WorkRequest{Order: r})
		}
	}
}

// -----------------  GatewayProtocolHandler Impl -----------------
func (gs *GatewaySvc) HandleShardComplete(req types.ShardCompleteReq) types.ShardCompleteResp {
	gs.locks.Lock("complete")
	defer gs.locks.Unlock("complete")

	logAndRespond := func(errMsg string, code uint64) types.ShardCompleteResp {
		log.Error(errMsg)
		return types.ShardCompleteResp{
			Code:    code,
			Message: errMsg,
		}
	}

	// query tx
	resultTx, err := gs.chainSvc.GetTx(gs.ctx, req.TxHash, req.Height)
	if err != nil {
		return logAndRespond(
			fmt.Sprintf("failed to get transaction %s(%v) at height(%d)", req.TxHash, req.Height, err),
			types.ErrorCodeInternalErr,
		)
	}

	if resultTx.TxResult.Code != 0 {
		return logAndRespond(
			fmt.Sprintf("tx %s failed with code %d", req.TxHash, resultTx.TxResult.Code),
			types.ErrorCodeInvalidTx,
		)
	}

	txb := tx.Tx{}
	err = txb.Unmarshal(resultTx.Tx)
	if err != nil {
		return logAndRespond(
			fmt.Sprintf("failed to decode tx(%s) body: %v", req.TxHash, err),
			types.ErrorCodeInvalidTx,
		)
	}

	m := saotypes.MsgComplete{}
	err = m.Unmarshal(txb.Body.Messages[0].Value)
	if err != nil {
		return logAndRespond(
			fmt.Sprintf("failed to decode tx(%s) body: %v", req.TxHash, err),
			types.ErrorCodeInvalidTx,
		)
	}

	order, err := gs.chainSvc.GetOrder(gs.ctx, m.OrderId)
	if err != nil {
		return logAndRespond(
			fmt.Sprintf("get order(%d) on chain error: %v", m.OrderId, err),
			types.ErrorCodeInternalErr,
		)
	}

	if order.Provider != gs.nodeAddress {
		return logAndRespond(
			fmt.Sprintf("order %d provider is %s, not %s", m.OrderId, order.Provider, gs.nodeAddress),
			types.ErrorCodeInvalidOrderProvider,
		)
	}

	shardCids := make(map[string]struct{})
	for key, shard := range order.Shards {
		if key == m.Creator {
			shardCids[shard.Cid] = struct{}{}
		}
	}
	if len(shardCids) <= 0 {
		return logAndRespond(
			fmt.Sprintf("order %d doesn't have shard provider %s", m.OrderId, m.Creator),
			types.ErrorCodeInvalidProvider,
		)
	}

	for _, cid := range req.Cids {
		if _, ok := shardCids[cid.String()]; !ok {
			return logAndRespond(
				fmt.Sprintf("%v is not in the given order %d", cid.String(), m.OrderId),
				types.ErrorCodeInvalidShardCid,
			)
		}
	}

	gs.locks.Lock(lockname(req.OrderId))
	defer gs.locks.Unlock(lockname(req.OrderId))

	orderInfo, err := utils.GetOrder(gs.ctx, gs.orderDs, req.DataId)
	if err != nil {
		return logAndRespond(
			fmt.Sprintf("get order on chain error: %v", err),
			types.ErrorCodeInternalErr,
		)
	}
	shardInfo := orderInfo.Shards[m.Creator]
	shardInfo.State = types.ShardStateCompleted
	shardInfo.CompleteHash = req.TxHash
	err = utils.SaveOrder(gs.ctx, gs.orderDs, orderInfo)
	if err != nil {
		log.Warn("put order %d error: %v", orderInfo.OrderId, err)
	}

	if orderInfo.State != types.OrderStateComplete && order.Status == saotypes.OrderCompleted {
		log.Debugf("complete channel done. order %d completes", orderInfo.OrderId)
		orderInfo.State = types.OrderStateComplete
		err = utils.SaveOrder(gs.ctx, gs.orderDs, orderInfo)
		if err != nil {
			log.Warn("put order %d error: %v", orderInfo.OrderId, err)
		}

		log.Debugf("unstage shard %s/%s/%v", gs.stagingPath, orderInfo.Owner, orderInfo.Cid)
		err := UnstageShard(gs.stagingPath, orderInfo.Owner, orderInfo.Cid.String())
		if err != nil {
			log.Warnf("unstage shard error: %v", err)
		}

		gs.completeResultChan <- orderInfo.DataId
	}
	return types.ShardCompleteResp{Code: 0}
}

func (gs *GatewaySvc) HandleShardStore(req types.ShardLoadReq) types.ShardLoadResp {
	resp := types.ShardLoadResp{
		OrderId:    req.OrderId,
		Cid:        req.Cid,
		RequestId:  req.RequestId,
		ResponseId: time.Now().UnixMilli(),
	}

	contentBytes, err := GetStagedShard(gs.stagingPath, req.Owner, req.Cid)
	if err != nil {
		resp.Code = types.ErrorCodeInternalErr
		resp.Message = fmt.Sprintf("Get staged shard(%v) error: %v", req.Cid, err)
		return resp
	}
	resp.Code = 0
	resp.Content = contentBytes
	return resp
}

func (gs *GatewaySvc) QueryMeta(ctx context.Context, req *types.MetadataProposal, height int64) (*types.Model, error) {
	res, err := gs.chainSvc.QueryMetadata(ctx, req, height)
	if err != nil {
		return nil, err
	}

	log.Debugf("QueryMeta succeed. meta=%v", res.Metadata)

	commit := res.Metadata.Commits[len(res.Metadata.Commits)-1]
	commitInfo, err := types.ParseMetaCommit(commit)
	if err != nil {
		return nil, types.Wrapf(types.ErrInvalidCommitInfo, "invalid commit information: %s", commit)
	}

	return &types.Model{
		DataId:   res.Metadata.DataId,
		Alias:    res.Metadata.Alias,
		GroupId:  res.Metadata.GroupId,
		Owner:    res.Metadata.Owner,
		OrderId:  res.Metadata.OrderId,
		Tags:     res.Metadata.Tags,
		Cid:      res.Metadata.Cid,
		Shards:   res.Shards,
		CommitId: commitInfo.CommitId,
		Commits:  res.Metadata.Commits,
		// Content: N/a,
		ExtendInfo: res.Metadata.ExtendInfo,
	}, nil
}

func (gs *GatewaySvc) FetchContent(ctx context.Context, req *types.MetadataProposal, meta *types.Model) (*FetchResult, error) {
	contentList := make([][]byte, len(meta.Shards))
	for key, shard := range meta.Shards {
		if contentList[shard.ShardId] != nil {
			continue
		}

		shardCid, err := cid.Decode(shard.Cid)
		if err != nil {
			return nil, types.Wrapf(types.ErrInvalidCid, "%s", shard.Cid)
		}

		var gp GatewayProtocol
		if key == gs.nodeAddress {
			gp = gs.gatewayProtocolMap["local"]
		} else {
			gp = gs.gatewayProtocolMap["stream"]
		}

		resp := gp.RequestShardLoad(ctx, types.ShardLoadReq{
			Cid:     shardCid,
			OrderId: meta.OrderId,
			Proposal: types.MetadataProposalCbor{
				Proposal: types.QueryProposal{
					Owner:           req.Proposal.Owner,
					Keyword:         req.Proposal.Keyword,
					GroupId:         req.Proposal.GroupId,
					KeywordType:     uint64(req.Proposal.KeywordType),
					LastValidHeight: req.Proposal.LastValidHeight,
					Gateway:         req.Proposal.Gateway,
					CommitId:        req.Proposal.CommitId,
					Version:         req.Proposal.Version,
				},
				JwsSignature: types.JwsSignature{
					Protected: req.JwsSignature.Protected,
					Signature: req.JwsSignature.Signature,
				},
			},
			RequestId:     time.Now().UnixMilli(),
			RelayProposal: gs.buildRelayProposal(ctx, gp, shard.Peer),
		}, shard.Peer, true)
		if resp.Code == 0 {
			contentList[shard.ShardId] = resp.Content
		} else {
			return nil, types.Wrapf(types.ErrFailuresResponsed, resp.Message)
		}
	}

	var content []byte
	for _, c := range contentList {
		content = append(content, c...)
	}

	contentCid, err := utils.CalculateCid(content)
	if err != nil {
		return nil, err
	}
	if contentCid.String() != meta.Cid {
		log.Errorf("cid mismatch, expected %s, but got %s", meta.Cid, contentCid.String())
	}

	match, err := regexp.Match("^"+types.Type_Prefix_File, []byte(meta.Alias))
	if err != nil {
		return nil, types.Wrapf(types.ErrInvalidAlias, "%s", meta.Alias)
	}

	if len(content) > gs.cfg.Cache.ContentLimit || match {
		// large size content should go through P2P channel

		path, err := homedir.Expand(gs.cfg.SaoHttpFileServer.HttpFileServerPath)
		if err != nil {
			return nil, types.Wrapf(types.ErrInvalidPath, "%s", gs.cfg.SaoHttpFileServer.HttpFileServerPath)
		}

		file, err := os.Create(filepath.Join(path, meta.DataId))
		if err != nil {
			return nil, types.Wrap(types.ErrInvalidPath, err)
		}

		_, err = file.Write([]byte(content))
		if err != nil {
			return nil, types.Wrap(types.ErrWriteFileFailed, err)
		}

		if gs.cfg.SaoIpfs.Enable {
			_, err = gs.storeManager.Store(ctx, contentCid, bytes.NewReader(content))
			if err != nil {
				return nil, types.Wrap(types.ErrStoreFailed, err)
			}
		}

		if len(content) > gs.cfg.Cache.ContentLimit {
			content = make([]byte, 0)
		}
	}

	return &FetchResult{
		Cid:     contentCid.String(),
		Content: content,
	}, nil
}

func (gs *GatewaySvc) buildRelayProposal(ctx context.Context, gp GatewayProtocol, peerInfos string) types.RelayProposalCbor {
	if gp.GetPeers(ctx) == "" {
		return types.RelayProposalCbor{
			Proposal:  types.RelayProposal{},
			Signature: make([]byte, 0),
		}
	}

	proposal := types.RelayProposal{
		NodeAddress:    gs.nodeAddress,
		LocalPeerId:    gs.localPeerId,
		RelayPeerIds:   gp.GetPeers(ctx),
		TargetPeerInfo: peerInfos,
	}

	buf := new(bytes.Buffer)
	err := proposal.MarshalCBOR(buf)
	if err != nil {
		log.Error(types.ErrMarshalFailed, err)
		return types.RelayProposalCbor{
			Proposal:  types.RelayProposal{},
			Signature: make([]byte, 0),
		}
	} else {
		signature, err := chain.SignByAddress(ctx, gs.keyringHome, gs.nodeAddress, buf.Bytes())
		log.Debug("keyringHome", gs.keyringHome)
		log.Debug("nodeAddress", gs.nodeAddress)
		if err != nil {
			log.Error(types.Wrap(types.ErrSignedFailed, err))
			return types.RelayProposalCbor{
				Proposal:  types.RelayProposal{},
				Signature: make([]byte, 0),
			}
		} else {
			return types.RelayProposalCbor{
				Proposal:  proposal,
				Signature: signature,
			}
		}
	}
}

func (gs *GatewaySvc) process(ctx context.Context, orderInfo types.OrderInfo) error {
	gs.locks.Lock(lockname(orderInfo.OrderId))
	defer gs.locks.Unlock(lockname(orderInfo.OrderId))

	if orderInfo.State == types.OrderStateTerminate {
		return nil
	}

	if orderInfo.State == types.OrderStateComplete {
		gs.completeResultChan <- orderInfo.DataId
		return nil
	}

	orderInfo.Tries++
	log.Infof("order dataid=%s tries=%d", orderInfo.DataId, orderInfo.Tries)
	if orderInfo.Tries >= 3 {
		orderInfo.State = types.OrderStateTerminate
		errMsg := fmt.Sprintf("order %d too many retries %d", orderInfo.OrderId, orderInfo.Tries)
		orderInfo.LastErr = errMsg
		e := utils.SaveOrder(ctx, gs.orderDs, orderInfo)
		if e != nil {
			log.Warn("put order %d error: %v", orderInfo.OrderId, e)
		}
		return nil
	}

	if orderInfo.ExpireHeight > 0 {
		latestHeight, err := gs.chainSvc.GetLastHeight(ctx)
		if err != nil {
			return err
		}

		if latestHeight > int64(orderInfo.ExpireHeight) {
			orderInfo.State = types.OrderStateTerminate
			errStr := fmt.Sprintf("order expired: latest=%d expireAt=%d", latestHeight, orderInfo.ExpireHeight)
			orderInfo.LastErr = errStr
			e := utils.SaveOrder(ctx, gs.orderDs, orderInfo)
			if e != nil {
				log.Warn("put order %d error: %v", orderInfo.OrderId, e)
			}
			return types.Wrapf(types.ErrExpiredOrder, errStr)
		}
	}

	if orderInfo.State < types.OrderStateComplete {
		log.Infof("assigning shards to nodes...")
		// assign shards to storage nodes

		log.Debugf("assigning order %d.", orderInfo.OrderId)
		for node, shard := range orderInfo.Shards {
			if shard.State != types.ShardStateCompleted {
				var gp GatewayProtocol
				if node == gs.nodeAddress {
					gp = gs.gatewayProtocolMap["local"]
				} else {
					gp = gs.gatewayProtocolMap["stream"]
				}
				req := types.ShardAssignReq{
					OrderId:      orderInfo.OrderId,
					TxHash:       orderInfo.OrderHash,
					DataId:       orderInfo.DataId,
					Assignee:     node,
					Height:       orderInfo.OrderHeight,
					AssignTxType: orderInfo.OrderTxType,
				}
				resp := gp.RequestShardAssign(ctx, req, shard.Peer)
				if resp.Code == 0 {
					shard.State = types.ShardStateNotified
					log.Infof("assigned order %d shard to node %s.", orderInfo.OrderId, node)
				} else {
					shard.State = types.ShardStateError
					log.Errorf("assigned order %d shards to node %s failed: %v", orderInfo.OrderId, node, resp.Message)
				}
			}
		}
		log.Debugf("assigned order %d done.", orderInfo.OrderId)

		err := utils.SaveOrder(ctx, gs.orderDs, orderInfo)
		if err != nil {
			return err
		}

		gs.locks.Lock("complete")
		if _, exists := gs.completeMap[orderInfo.DataId]; !exists {
			gs.completeMap[orderInfo.DataId] = 0
		}
		gs.locks.Unlock("complete")
	}

	return nil
}

func (gs *GatewaySvc) CommitModel(ctx context.Context, clientProposal *types.OrderStoreProposal, orderId uint64, content []byte) (*CommitResult, error) {
	// stage order data.
	orderProposal := clientProposal.Proposal
	stagePath, err := StageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid, content)
	if err != nil {
		return nil, err
	}

	proposalBytes, err := clientProposal.Proposal.Marshal()
	if err != nil {
		return nil, err
	}
	signatureBytes, err := clientProposal.JwsSignature.Marshal()
	if err != nil {
		return nil, err
	}
	cid, err := cid.Decode(clientProposal.Proposal.Cid)
	if err != nil {
		return nil, err
	}
	orderInfo := types.OrderInfo{
		State:     types.OrderStateStaged,
		StagePath: stagePath,
		DataId:    clientProposal.Proposal.DataId,
		OrderId:   orderId,
		Owner:     clientProposal.Proposal.Owner,
		Cid:       cid,
	}
	err = utils.SaveOrder(ctx, gs.orderDs, orderInfo)
	if err != nil {
		return nil, err
	}

	var proposal saotypes.Proposal
	err = proposal.Unmarshal(proposalBytes)
	if err != nil {
		return nil, err
	}

	var txHash string
	var shards []*saotypes.ShardMeta
	var txType types.AssignTxType
	var height int64
	if orderInfo.State < types.OrderStateReady {
		if orderInfo.OrderId == 0 {
			var signature saotypes.JwsSignature
			err := signature.Unmarshal(signatureBytes)
			if err != nil {
				return nil, err
			}

			clientProposal := types.OrderStoreProposal{
				Proposal:     proposal,
				JwsSignature: signature,
			}
			var resp saotypes.MsgStoreResponse
			resp, txHash, height, err = gs.chainSvc.StoreOrder(ctx, gs.nodeAddress, &clientProposal)
			if err != nil {
				return nil, err
			}
			shards = resp.Shards
			txType = types.AssignTxTypeStore
			log.Infof("StoreOrder tx succeed. orderId=%d tx=%s shards=%v", resp.OrderId, txHash, resp.Shards)

			orderInfo.OrderId = resp.OrderId
		} else {
			log.Debugf("Sending OrderReady... orderId=%d", orderInfo.OrderId)
			var resp saotypes.MsgReadyResponse
			resp, txHash, height, err = gs.chainSvc.OrderReady(ctx, gs.nodeAddress, orderInfo.OrderId)
			if err != nil {
				return nil, err
			}
			shards = resp.Shards
			txType = types.AssignTxTypeReady
			log.Infof("OrderReady tx succeed. orderId=%d tx=%s", resp.OrderId, txHash)
			log.Infof("OrderReady tx succeed. shards=%v", resp.Shards)

			orderInfo.OrderId = resp.OrderId
		}
		orderInfo.OrderHash = txHash
		orderInfo.OrderHeight = height
		orderInfo.OrderTxType = txType
		orderInfo.State = types.OrderStateReady
		orderInfo.Shards = make(map[string]types.OrderShardInfo)
		for _, s := range shards {
			orderInfo.Shards[s.Sp] = types.OrderShardInfo{
				ShardId:  s.ShardId,
				Peer:     s.Peer,
				Cid:      s.Cid,
				Provider: s.Provider,
				State:    types.ShardStateAssigned,
			}
		}

		order, err := gs.chainSvc.GetOrder(ctx, orderInfo.OrderId)
		if err == nil {
			orderInfo.ExpireHeight = uint64(order.Expire)
		} else {
			log.Warn("chain get order err: ", err)
		}
		err = utils.SaveOrder(ctx, gs.orderDs, orderInfo)
		if err != nil {
			return nil, err
		}
	}

	gs.schedule <- &WorkRequest{Order: orderInfo}

	// TODO: wsevent
	//err = gs.chainSvc.UnsubscribeOrderComplete(ctx, orderId)
	//if err != nil {
	//	log.Error(err)
	//} else {
	//	log.Debugf("UnsubscribeOrderComplete")
	//}

	oi, err := utils.GetOrder(ctx, gs.orderDs, orderInfo.DataId)
	if err != nil {
		return nil, err
	}

	return &CommitResult{
		OrderId: oi.OrderId,
		DataId:  oi.DataId,
		Cid:     oi.Cid.String(),
	}, nil
}

func (gs *GatewaySvc) TerminateOrder(ctx context.Context, req *types.OrderTerminateProposal) error {
	_, err := gs.chainSvc.TerminateOrder(ctx, gs.nodeAddress, *req)
	if err != nil {
		return err
	}

	return nil
}

func (gs *GatewaySvc) RenewOrder(ctx context.Context, req *types.OrderRenewProposal) (map[string]string, error) {
	_, results, err := gs.chainSvc.RenewOrder(ctx, gs.nodeAddress, *req)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (gs *GatewaySvc) UpdateModelPermission(ctx context.Context, req *types.PermissionProposal) error {
	_, err := gs.chainSvc.UpdatePermission(ctx, gs.nodeAddress, req)
	if err != nil {
		return err
	}

	return nil
}

func (gs *GatewaySvc) Stop(ctx context.Context) error {
	log.Info("stopping gateway service...")

	var err error
	for k, p := range gs.gatewayProtocolMap {
		err = p.Stop(ctx)
		if err != nil {
			log.Errorf("stopping %s gateway protocol failed: %v", k, err)
		} else {
			log.Infof("%s gateway protocol stopped.", k)
		}
	}

	log.Info("close complete result chan...")
	close(gs.completeResultChan)

	return nil
}

func (gs *GatewaySvc) OrderStatus(ctx context.Context, id string) (types.OrderInfo, error) {
	return utils.GetOrder(ctx, gs.orderDs, id)
}

func (gs *GatewaySvc) getOrderKeys(ctx context.Context) ([]types.OrderKey, error) {
	index, err := utils.GetOrderIndex(ctx, gs.orderDs)
	if err != nil {
		return nil, err
	}

	return index.Alls, nil
}

func (gs *GatewaySvc) OrderList(ctx context.Context) ([]types.OrderInfo, error) {
	keys, err := gs.getOrderKeys(ctx)
	if err != nil {
		return nil, err
	}

	var orderInfos []types.OrderInfo
	for _, orderId := range keys {
		orderInfo, err := utils.GetOrder(ctx, gs.orderDs, orderId.DataId)
		if err != nil {
			return nil, err
		}
		orderInfos = append(orderInfos, orderInfo)
	}
	return orderInfos, nil
}

func (gs *GatewaySvc) OrderFix(ctx context.Context, dataId string) error {
	orderInfo, err := utils.GetOrder(ctx, gs.orderDs, dataId)
	if err != nil {
		return err
	}

	gs.schedule <- &WorkRequest{Order: orderInfo}
	return nil
}

func (gs *GatewaySvc) getPendingOrders(ctx context.Context) ([]types.OrderInfo, error) {
	orderKeys, err := gs.getOrderKeys(ctx)
	if err != nil {
		return nil, err
	}

	var orders []types.OrderInfo
	for _, orderKey := range orderKeys {
		order, err := utils.GetOrder(ctx, gs.orderDs, orderKey.DataId)
		if err != nil {
			return nil, err
		}
		if order.State != types.OrderStateComplete {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func lockname(orderId uint64) string {
	return fmt.Sprintf("lk-order-%d", orderId)
}
