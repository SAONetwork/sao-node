package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sao-node/chain"
	"sao-node/node/config"
	"sao-node/node/transport"
	"sao-node/store"
	"sao-node/types"
	"sao-node/utils"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"golang.org/x/xerrors"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
)

var log = logging.Logger("gateway")

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
	OrderStatus(ctx context.Context, orderId uint64) (types.OrderInfo, error)
	OrderList(ctx context.Context) ([]types.OrderInfo, error)
}

type GatewaySvc struct {
	ctx                context.Context
	chainSvc           *chain.ChainSvc
	shardStreamHandler *ShardStreamHandler
	storeManager       *store.StoreManager
	nodeAddress        string
	stagingPath        string
	cfg                *config.Node
	notifyChan         map[string]chan interface{}
	completeChan       map[uint64]chan struct{}
	orderDs            datastore.Batching
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
) *GatewaySvc {
	cs := &GatewaySvc{
		ctx:          ctx,
		chainSvc:     chainSvc,
		storeManager: storeManager,
		nodeAddress:  nodeAddress,
		stagingPath:  cfg.Transport.StagingPath,
		cfg:          cfg,
		notifyChan:   notifyChan,
		completeChan: make(map[uint64]chan struct{}),
		orderDs:      orderDs,
	}
	cs.shardStreamHandler = NewShardStreamHandler(ctx, host, cfg.Transport.StagingPath, chainSvc, nodeAddress, cs.completeChan)

	if c, exists := notifyChan[types.ShardCompleteProtocol]; exists {
		go cs.subscribeShardComplete(ctx, c)
	}

	return cs
}

func (ss *GatewaySvc) subscribeShardComplete(ctx context.Context, shardChan chan interface{}) {
	for {
		select {
		case t, ok := <-shardChan:
			if !ok {
				return
			}
			// process
			resp := ss.handleShardComplete(t.(types.ShardCompleteReq))
			if resp.Code != 0 {
				log.Errorf(resp.Message)
			}
		case <-ctx.Done():
			return
		}
	}
}

// how to merge this function in stream handler??
func (ss *GatewaySvc) handleShardComplete(req types.ShardCompleteReq) types.ShardCompleteResp {
	if req.Code != 0 {
		// TODO: notify channel that storage node can't handle this shard.
		log.Debugf("storage node can't handle order %d shards %v: %s", req.OrderId, req.Cids, req.Message)
		return types.ShardCompleteResp{Code: 0}
	}

	// query tx
	resultTx, err := ss.chainSvc.GetTx(ss.ctx, req.TxHash, req.Height)
	if err != nil {
		return types.ShardCompleteResp{
			Code:    types.ErrorCodeInternalErr,
			Message: fmt.Sprintf("internal error: %v", err),
		}
	}
	if resultTx.TxResult.Code == 0 {
		txb := tx.Tx{}
		err = txb.Unmarshal(resultTx.Tx)
		if err != nil {
			return types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidTx,
				Message: fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			}
		}

		m := saotypes.MsgComplete{}
		err = m.Unmarshal(txb.Body.Messages[0].Value)
		if err != nil {
			return types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidTx,
				Message: fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			}
		}

		order, err := ss.chainSvc.GetOrder(ss.ctx, m.OrderId)
		if err != nil {
			return types.ShardCompleteResp{
				Code:    types.ErrorCodeInternalErr,
				Message: fmt.Sprintf("internal error: %v", err),
			}
		}

		if order.Provider != ss.nodeAddress {
			return types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidOrderProvider,
				Message: fmt.Sprintf("order %d provider is %s, not %s", m.OrderId, order.Provider, ss.nodeAddress),
			}
		}

		shardCids := make(map[string]struct{})
		for key, shard := range order.Shards {
			if key == m.Creator {
				shardCids[shard.Cid] = struct{}{}
			}
		}
		if len(shardCids) <= 0 {
			return types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidProvider,
				Message: fmt.Sprintf("order %d doesn't have shard provider %s", m.OrderId, m.Creator),
			}
		}

		for _, cid := range req.Cids {
			if _, ok := shardCids[cid.String()]; !ok {
				return types.ShardCompleteResp{
					Code:    types.ErrorCodeInvalidShardCid,
					Message: fmt.Sprintf("%v is not in the given order %d", cid.String(), m.OrderId),
				}
			}
		}

		orderInfo, err := utils.GetOrder(ss.ctx, ss.orderDs, order.Id)
		if err != nil {
			log.Warnf("get order %d error: %v", err)
		}
		shardInfo := orderInfo.Shards[m.Creator]
		shardInfo.State = types.ShardStateCompleted
		shardInfo.CompleteHash = req.TxHash
		err = utils.PutOrder(ss.ctx, ss.orderDs, orderInfo)
		if err != nil {
			log.Warn("put order %d error: %v", orderInfo.OrderId, err)
		}

		if order.Status == saotypes.OrderCompleted {
			// update channel.
			ss.completeChan[m.OrderId] <- struct{}{}
		}
		return types.ShardCompleteResp{Code: 0}
	} else {
		// respond storage node to re-handle this shard.
		return types.ShardCompleteResp{
			Code:    types.ErrorCodeInvalidTx,
			Message: fmt.Sprintf("tx %s failed with code %d", req.TxHash, resultTx.TxResult.Code),
		}
	}
}

func (gs *GatewaySvc) QueryMeta(ctx context.Context, req *types.MetadataProposal, height int64) (*types.Model, error) {
	res, err := gs.chainSvc.QueryMetadata(ctx, req, height)
	if err != nil {
		return nil, err
	}

	log.Debugf("QueryMeta succeed. meta=%v", res.Metadata)

	commit := res.Metadata.Commits[len(res.Metadata.Commits)-1]
	commitInfo := strings.Split(commit, "\032")
	if len(commitInfo) != 2 || len(commitInfo[1]) == 0 {
		return nil, xerrors.Errorf("invalid commit information: %s", commit)
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
		CommitId: commitInfo[0],
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
			return nil, err
		}

		var shardContent []byte
		if key == gs.nodeAddress {
			// local shard
			if gs.storeManager == nil {
				return nil, xerrors.Errorf("local store manager not found")
			}
			reader, err := gs.storeManager.Get(ctx, shardCid)
			if err != nil {
				return nil, err
			}
			shardContent, err = io.ReadAll(reader)
			if err != nil {
				return nil, err
			}
		} else {
			// remote shard
			for _, peerInfo := range strings.Split(shard.Peer, ",") {

				_, err := multiaddr.NewMultiaddr(peerInfo)

				if err != nil {
					return nil, err
				}

				if strings.Contains(peerInfo, "udp") || strings.Contains(peerInfo, "127.0.0.1") {
					continue
				}

				shardContent, err = gs.shardStreamHandler.Fetch(req, peerInfo, meta.OrderId, shardCid)
				if err != nil {
					return nil, err
				}
				break
			}
		}
		contentList[shard.ShardId] = shardContent
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
		return nil, err
	}

	if len(content) > gs.cfg.Cache.ContentLimit || match {
		// large size content should go through P2P channel

		path, err := homedir.Expand(gs.cfg.SaoHttpFileServer.HttpFileServerPath)
		if err != nil {
			return nil, err
		}

		file, err := os.Create(filepath.Join(path, meta.DataId))
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}

		_, err = file.Write([]byte(content))
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}

		if gs.cfg.SaoIpfs.Enable {
			_, err = gs.storeManager.Store(ctx, contentCid, bytes.NewReader(content))
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
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

func (gs *GatewaySvc) execOrder(ctx context.Context, clientProposal *types.OrderStoreProposal, orderInfo types.OrderInfo) error {
	var err error

	// send tx
	var txHash string
	var shards map[string]*saotypes.ShardMeta
	var txType types.AssignTxType
	var height int64
	if orderInfo.State < types.OrderStateReady {
		if orderInfo.OrderId == 0 {
			resp, txId, h, err := gs.chainSvc.StoreOrder(ctx, gs.nodeAddress, clientProposal)
			if err != nil {
				orderInfo.LastErr = err.Error()
				e := utils.PutOrder(ctx, gs.orderDs, orderInfo)
				if e != nil {
					log.Warn("put order %d error: %v", orderInfo.OrderId, e)
				}
				return err
			}
			shards = resp.Shards
			txHash = txId
			txType = types.AssignTxTypeStore
			height = h
			log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", resp.OrderId, txId)
			log.Infof("StoreOrder tx succeed. shards=%v", resp.Shards)

			orderInfo.OrderHash = txId
			orderInfo.OrderId = resp.OrderId
		} else {
			log.Debugf("Sending OrderReady... orderId=%d", orderInfo.OrderId)
			resp, txId, h, err := gs.chainSvc.OrderReady(ctx, gs.nodeAddress, orderInfo.OrderId)
			if err != nil {
				return err
			}
			shards = resp.Shards
			txHash = txId
			height = h
			txType = types.AssignTxTypeReady
			log.Infof("OrderReady tx succeed. orderId=%d tx=%s", resp.OrderId, txId)
			log.Infof("OrderReady tx succeed. shards=%v", resp.Shards)

			orderInfo.ReadyHash = txId
		}
		orderInfo.State = types.OrderStateReady
		orderInfo.Shards = make(map[string]types.ShardInfo)
		for node, s := range shards {
			orderInfo.Shards[node] = types.ShardInfo{
				ShardId:  s.ShardId,
				Peer:     s.Peer,
				Cid:      s.Cid,
				Provider: s.Provider,
				State:    types.ShardStateAssigned,
			}
		}
		err = utils.PutOrder(ctx, gs.orderDs, orderInfo)
		if err != nil {
			return err
		}
	}

	if orderInfo.State < types.OrderStateComplete {
		log.Infof("assigning shards to nodes...")
		// assign shards to storage nodes

		for node, shard := range orderInfo.Shards {
			if shard.State != types.ShardStateCompleted {
				var shardAssignReq = types.ShardAssignReq{
					OrderId:      orderInfo.OrderId,
					TxHash:       txHash,
					Assignee:     node,
					Height:       height,
					AssignTxType: txType,
				}
				if node == gs.nodeAddress {
					if c, exists := gs.notifyChan[types.ShardAssignProtocol]; exists {
						c <- shardAssignReq
						shard.State = types.ShardStateNotified
					} else {
						log.Errorf("assign order %d shards to node %s failed: %v", orderInfo.OrderId, node, "no channel defined")
						shard.State = types.ShardStateError
						continue
					}
				} else {
					resp := types.ShardAssignResp{}
					err = transport.HandleRequest(
						ctx,
						shard.Peer,
						gs.shardStreamHandler.host,
						types.ShardAssignProtocol,
						&shardAssignReq,
						&resp,
					)
					if err != nil {
						log.Errorf("assign order %d shards to node %s failed: %v", orderInfo.OrderId, node, err)
						shard.State = types.ShardStateNotified
						continue
					}
					if resp.Code == 0 {
						shard.State = types.ShardStateNotified
						log.Infof("assigned order %d shard to node %s.", orderInfo.OrderId, node)
					} else {
						log.Errorf("assigned order %d shards to node %s failed: %v", orderInfo.OrderId, node, resp.Message)
						shard.State = types.ShardStateError
					}
				}
			}
		}
		err = utils.PutOrder(ctx, gs.orderDs, orderInfo)
		if err != nil {
			return err
		}

		log.Infof("waiting for all nodes order %d shard completion.", orderInfo.OrderId)
		if _, exists := gs.completeChan[orderInfo.OrderId]; !exists {
			gs.completeChan[orderInfo.OrderId] = make(chan struct{})
		}

		<-gs.completeChan[orderInfo.OrderId]
		log.Debugf("complete channel done. order %d completes", orderInfo.OrderId)
		orderInfo.State = types.OrderStateComplete
		err = utils.PutOrder(ctx, gs.orderDs, orderInfo)
		if err != nil {
			log.Warn("put order %d error: %v", orderInfo.OrderId, err)
		}
		close(gs.completeChan[orderInfo.OrderId])
		delete(gs.completeChan, orderInfo.OrderId)

		log.Debugf("unstage shard %s/%s/%v", gs.stagingPath, clientProposal.Proposal.Owner, clientProposal.Proposal.Cid)
		err = UnstageShard(gs.stagingPath, clientProposal.Proposal.Owner, clientProposal.Proposal.Cid)
		if err != nil {
			log.Warn("unstage shard error: %v", err)
		}
	}
	return nil
}

func (gs *GatewaySvc) CommitModel(ctx context.Context, clientProposal *types.OrderStoreProposal, orderId uint64, content []byte) (*CommitResult, error) {
	orderProposal := clientProposal.Proposal
	stagePath, err := StageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid, content)
	if err != nil {
		return nil, err
	}

	orderInfo, err := utils.GetOrder(ctx, gs.orderDs, orderId)
	if err != nil {
		return nil, err
	}
	if orderInfo.OrderId == 0 {
		orderInfo = types.OrderInfo{
			State:     types.OrderStateStaged,
			StagePath: stagePath,
		}
		err = utils.PutOrder(ctx, gs.orderDs, orderInfo)
		if err != nil {
			return nil, err
		}
	} else {
		orderInfo.StagePath = stagePath
		orderInfo.State = types.OrderStateStaged
		err = utils.PutOrder(ctx, gs.orderDs, orderInfo)
		if err != nil {
			return nil, err
		}
	}

	// case <-time.After(chain.Blocktime * time.Duration(clientProposal.Proposal.Timeout)):
	ch := make(chan string, 1)
	go func() {
		err = gs.execOrder(ctx, clientProposal, orderInfo)
		if err != nil {
			ch <- err.Error()
		} else {
			ch <- "ok"
		}
	}()

	timeout := false
	select {
	case s := <-ch:
		if s != "ok" {
			return nil, xerrors.Errorf(s)
		}
		break
	case <-time.After(chain.Blocktime * time.Duration(clientProposal.Proposal.Timeout)):
		timeout = true
	}

	// TODO: wsevent
	//err = gs.chainSvc.UnsubscribeOrderComplete(ctx, orderId)
	//if err != nil {
	//	log.Error(err)
	//} else {
	//	log.Debugf("UnsubscribeOrderComplete")
	//}

	if timeout {
		// TODO: timeout handling
		return nil, errors.Errorf("process order %d timeout.", orderId)
	} else {
		order, err := gs.chainSvc.GetOrder(ctx, orderId)
		if err != nil {
			return nil, err
		}
		log.Debugf("order %d complete: dataId=%s", orderId, order.Metadata.DataId)

		shards := make(map[string]*saotypes.ShardMeta, 0)
		for peer, shard := range order.Shards {
			meta := saotypes.ShardMeta{
				ShardId:  shard.Id,
				Peer:     peer,
				Cid:      shard.Cid,
				Provider: order.Provider,
			}
			shards[peer] = &meta
		}

		return &CommitResult{
			OrderId: order.Metadata.OrderId,
			DataId:  order.Metadata.DataId,
			Commit:  order.Metadata.Commit,
			Commits: order.Metadata.Commits,
			Shards:  shards,
			Cid:     orderProposal.Cid,
		}, nil
	}
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
	log.Info("stopping order service...")
	gs.shardStreamHandler.Stop(ctx)

	return nil
}

func (gs *GatewaySvc) OrderStatus(ctx context.Context, orderId uint64) (types.OrderInfo, error) {
	return utils.GetOrder(ctx, gs.orderDs, orderId)
}

func (gs *GatewaySvc) OrderList(ctx context.Context) ([]types.OrderInfo, error) {
	key := datastore.NewKey("order_stats")
	exists, err := gs.orderDs.Has(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []types.OrderInfo{}, nil
	}
	data, err := gs.orderDs.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var orderStats types.OrderStats
	err = orderStats.UnmarshalCBOR(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var orderInfos []types.OrderInfo
	for _, orderId := range orderStats.All {
		orderInfo, err := utils.GetOrder(ctx, gs.orderDs, orderId)
		if err != nil {
			return nil, err
		}
		orderInfos = append(orderInfos, orderInfo)
	}
	return orderInfos, nil
}

func (gs *GatewaySvc) OrderFix(ctx context.Context, orderId uint64) (types.OrderInfo, error) {

}
