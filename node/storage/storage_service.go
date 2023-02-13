package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sao-node/chain"
	"sao-node/store"
	"sao-node/types"
	"sao-node/utils"
	"time"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"golang.org/x/xerrors"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"

	sid "github.com/SaoNetwork/sao-did/sid"
	logging "github.com/ipfs/go-log/v2"

	saodid "github.com/SaoNetwork/sao-did"
	saodidtypes "github.com/SaoNetwork/sao-did/types"

	"github.com/libp2p/go-libp2p/core/host"
)

var log = logging.Logger("storage")

type StoreSvc struct {
	nodeAddress        string
	chainSvc           *chain.ChainSvc
	taskChan           chan types.ShardInfo
	host               host.Host
	stagingPath        string
	storeManager       *store.StoreManager
	ctx                context.Context
	orderDs            datastore.Batching
	storageProtocolMap map[string]StorageProtocol
}

func NewStoreService(
	ctx context.Context,
	nodeAddress string,
	chainSvc *chain.ChainSvc,
	host host.Host,
	stagingPath string,
	storeManager *store.StoreManager,
	notifyChan map[string]chan interface{},
	orderDs datastore.Batching,
) (*StoreSvc, error) {
	ss := &StoreSvc{
		nodeAddress:  nodeAddress,
		chainSvc:     chainSvc,
		taskChan:     make(chan types.ShardInfo),
		host:         host,
		stagingPath:  stagingPath,
		storeManager: storeManager,
		ctx:          ctx,
		orderDs:      orderDs,
	}

	ss.storageProtocolMap = make(map[string]StorageProtocol)
	ss.storageProtocolMap["local"] = NewLocalStorageProtocol(
		ctx,
		notifyChan,
		stagingPath,
		ss,
	)
	ss.storageProtocolMap["stream"] = NewStreamStorageProtocol(host, ss)

	// wsevent way to receive shard assign
	//if err := ss.chainSvc.SubscribeShardTask(ctx, ss.nodeAddress, ss.taskChan); err != nil {
	//	return nil, err
	//}

	go ss.processIncompleteShards(ctx)

	return ss, nil
}

func (ss *StoreSvc) processIncompleteShards(ctx context.Context) {
	log.Info("processing pending shards...")
	pendings, err := ss.getPendingShardList(ctx)
	if err != nil {
		log.Errorf("process pending shards error: %v", err)
	}
	for _, p := range pendings {
		ss.taskChan <- p
	}
}

func (ss *StoreSvc) HandleShardLoad(req types.ShardLoadReq) types.ShardLoadResp {
	logAndRespond := func(code uint64, errMsg string) types.ShardLoadResp {
		log.Error(errMsg)
		return types.ShardLoadResp{
			Code:       code,
			Message:    errMsg,
			OrderId:    req.OrderId,
			Cid:        req.Cid,
			RequestId:  req.RequestId,
			ResponseId: time.Now().UnixMilli(),
		}
	}

	didManager, err := saodid.NewDidManagerWithDid(req.Proposal.Proposal.Owner, ss.getSidDocFunc())
	if err != nil {
		return logAndRespond(types.ErrorCodeInternalErr, fmt.Sprintf("invalid did: %v", err))
	}

	p := saotypes.QueryProposal{
		Owner:           req.Proposal.Proposal.Owner,
		Keyword:         req.Proposal.Proposal.Keyword,
		GroupId:         req.Proposal.Proposal.GroupId,
		KeywordType:     uint32(req.Proposal.Proposal.KeywordType),
		LastValidHeight: req.Proposal.Proposal.LastValidHeight,
		Gateway:         req.Proposal.Proposal.Gateway,
		CommitId:        req.Proposal.Proposal.CommitId,
		Version:         req.Proposal.Proposal.Version,
	}

	proposalBytes, err := p.Marshal()
	if err != nil {
		return logAndRespond(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("marshal error: %v", err),
		)
	}

	_, err = didManager.VerifyJWS(saodidtypes.GeneralJWS{
		Payload: base64url.Encode(proposalBytes),
		Signatures: []saodidtypes.JwsSignature{
			saodidtypes.JwsSignature(req.Proposal.JwsSignature),
		},
	})

	if err != nil {
		return logAndRespond(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("verify client order proposal signature failed: %v", err),
		)
	}

	lastHeight, err := ss.chainSvc.GetLastHeight(ss.ctx)
	if err != nil {
		return logAndRespond(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("get chain height error: %v", err),
		)
	}

	if req.Proposal.Proposal.LastValidHeight < uint64(lastHeight) {
		return logAndRespond(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("invalid query, LastValidHeight:%d > now:%d", req.Proposal.Proposal.LastValidHeight, lastHeight),
		)
	}

	log.Debugf("Get %v", req.Cid)
	reader, err := ss.storeManager.Get(ss.ctx, req.Cid)
	if err != nil {
		return logAndRespond(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("get %v from store error: %v", req.Cid, err),
		)
	}
	shardContent, err := io.ReadAll(reader)
	if err != nil {
		return logAndRespond(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("get %v from store error: %v", req.Cid, err),
		)
	}

	return types.ShardLoadResp{
		OrderId:    req.OrderId,
		Cid:        req.Cid,
		Content:    shardContent,
		RequestId:  req.RequestId,
		ResponseId: time.Now().UnixMilli(),
	}
}

func (ss *StoreSvc) HandleShardAssign(req types.ShardAssignReq) types.ShardAssignResp {
	logAndRespond := func(code uint64, errMsg string) types.ShardAssignResp {
		log.Error(errMsg)
		return types.ShardAssignResp{
			Code:    code,
			Message: errMsg,
		}
	}

	// validate request
	if req.Assignee != ss.nodeAddress {
		return logAndRespond(
			types.ErrorCodeInvalidShardAssignee,
			fmt.Sprintf("shard assignee is %s, but current node is %s", req.Assignee, ss.nodeAddress),
		)
	}

	resultTx, err := ss.chainSvc.GetTx(ss.ctx, req.TxHash, req.Height)
	if err != nil {
		return logAndRespond(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("internal error: %v", err),
		)
	}

	if resultTx.TxResult.Code == 0 {
		txb := tx.Tx{}
		err = txb.Unmarshal(resultTx.Tx)
		if err != nil {
			return logAndRespond(
				types.ErrorCodeInvalidTx,
				fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			)
		}

		// validate tx
		if req.AssignTxType == types.AssignTxTypeStore {
			m := saotypes.MsgStore{}
			err = m.Unmarshal(txb.Body.Messages[0].Value)
		} else {
			m := saotypes.MsgReady{}
			err = m.Unmarshal(txb.Body.Messages[0].Value)
		}
		if err != nil {
			return logAndRespond(
				types.ErrorCodeInvalidTx,
				fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			)
		}

		order, err := ss.chainSvc.GetOrder(ss.ctx, req.OrderId)
		if err != nil {
			return logAndRespond(
				types.ErrorCodeInternalErr,
				fmt.Sprintf("internal error: %v", err),
			)
		}

		var shardCids []string
		for key, shard := range order.Shards {
			if key == ss.nodeAddress {
				shardCids = append(shardCids, shard.Cid)
			}
		}
		if len(shardCids) <= 0 {
			return logAndRespond(
				types.ErrorCodeInvalidProvider,
				fmt.Sprintf("order %d doesn't have shard provider %s", req.OrderId, ss.nodeAddress),
			)
		}
		for _, shardCid := range shardCids {
			cid, err := cid.Decode(shardCid)
			if err != nil {
				return logAndRespond(
					types.ErrorCodeInvalidShardCid,
					fmt.Sprintf("invalid cid %s", shardCid),
				)
			}

			shardInfo, _ := utils.GetShard(ss.ctx, ss.orderDs, req.OrderId, cid)
			if (types.ShardInfo{} == shardInfo) {
				shardInfo = types.ShardInfo{
					Owner:          order.Owner,
					OrderId:        req.OrderId,
					Gateway:        order.Provider,
					Cid:            cid,
					DataId:         req.DataId,
					OrderOperation: fmt.Sprintf("%d", order.Operation),
					ShardOperation: fmt.Sprintf("%d", order.Operation),
					State:          types.ShardStateValidated,
				}
				err = utils.SaveShard(ss.ctx, ss.orderDs, shardInfo)
				if err != nil {
					// do not throw error, the best case is storage node handle shard again.
					log.Warn("put shard order=%d cid=%v error: %v", shardInfo.OrderId, shardInfo.Cid, err)
				}
			}
			ss.taskChan <- shardInfo
		}
		return types.ShardAssignResp{Code: 0}
	} else {
		return logAndRespond(
			types.ErrorCodeInvalidTx,
			fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
		)
	}
}

func (ss *StoreSvc) Start(ctx context.Context) error {
	for {
		select {
		case t, ok := <-ss.taskChan:
			if !ok {
				return nil
			}
			err := ss.process(ctx, t)
			if err != nil {
				// TODO: retry mechanism
				log.Error(err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (ss *StoreSvc) process(ctx context.Context, task types.ShardInfo) error {
	log.Infof("start processing: order id=%d gateway=%s shard_cid=%v", task.OrderId, task.Gateway, task.Cid)

	var err error

	sp, peerInfo, err := ss.getStorageProtocolAndPeer(ctx, task.Gateway)
	if err != nil {
		ss.updateShardError(task, err)
		return err
	}

	if task.State < types.ShardStateStored {
		// check if it's a renew order(Operation is 3)
		if task.OrderOperation != "3" || task.ShardOperation != "3" {
			resp := sp.RequestShardStore(ctx, types.ShardLoadReq{
				Owner:   task.Owner,
				OrderId: task.OrderId,
				Cid:     task.Cid,
			}, peerInfo)
			if resp.Code != 0 {
				ss.updateShardError(task, xerrors.Errorf(resp.Message))
				return xerrors.Errorf(resp.Message)
			} else {
				cid, _ := utils.CalculateCid(resp.Content)
				log.Debugf("ipfs cid %v, task cid %v, order id %v", cid, task.Cid, task.OrderId)
				if cid.String() != task.Cid.String() {
					ss.updateShardError(task, err)
					return types.Wrapf(types.ErrInvalidCid, "ipfs cid %v != task cid %v", cid, task.Cid)
				}
			}

			// store to backends
			_, err = ss.storeManager.Store(ctx, task.Cid, bytes.NewReader(resp.Content))
			if err != nil {
				ss.updateShardError(task, err)
				return types.Wrap(types.ErrStoreFailed, err)
			}
			task.Size = uint64(len(resp.Content))
		} else {
			// make sure the data is still there
			isExist := ss.storeManager.IsExist(ctx, task.Cid)
			if !isExist {
				ss.updateShardError(task, err)
				return types.Wrapf(types.ErrDataMissing, "shard with cid %s not found", task.Cid)
			}
		}
		task.State = types.ShardStateStored
		err = utils.SaveShard(ctx, ss.orderDs, task)
		if err != nil {
			log.Warnf("put shard order=%d cid=%v error: %v", task.OrderId, task.Cid, err)
		}
	}

	if task.State < types.ShardStateTxSent {
		txHash, height, err := ss.chainSvc.CompleteOrder(ctx, ss.nodeAddress, task.OrderId, task.Cid, int32(task.Size))
		if err != nil {
			ss.updateShardError(task, err)
			return err
		}
		log.Infof("Complete order succeed: txHash: %s, OrderId: %d, cid: %s", txHash, task.OrderId, task.Cid)

		task.State = types.ShardStateComplete
		task.CompleteHash = txHash
		task.CompleteHeight = height
		err = utils.SaveShard(ss.ctx, ss.orderDs, task)
		if err != nil {
			log.Warnf("put shard order=%d cid=%v error: %v", task.OrderId, task.Cid, err)
		}
	}

	resp := sp.RequestShardComplete(ctx, types.ShardCompleteReq{
		OrderId: task.OrderId,
		DataId:  task.DataId,
		Cids:    []cid.Cid{task.Cid},
		Height:  task.CompleteHeight,
		TxHash:  task.CompleteHash,
	}, peerInfo)
	if resp.Code != 0 {
		ss.updateShardError(task, xerrors.Errorf(resp.Message))
		// return xerrors.Errorf(resp.Message)
	}
	if task.State < types.ShardStateComplete {
		task.State = types.ShardStateComplete
		err = utils.SaveShard(ss.ctx, ss.orderDs, task)
		if err != nil {
			log.Warnf("put shard order=%d cid=%v error: %v", task.OrderId, task.Cid, err)
		}
	}
	return nil
}

func (ss *StoreSvc) Stop(ctx context.Context) error {
	// TODO: wsevent
	//if err := ss.chainSvc.UnsubscribeShardTask(ctx, ss.nodeAddress); err != nil {
	//	return err
	//}
	log.Info("stopping storage service...")
	close(ss.taskChan)

	var err error
	for k, p := range ss.storageProtocolMap {
		err = p.Stop(ctx)
		if err != nil {
			log.Error("stopping %s storage protocol failed: %v", k, err)
		} else {
			log.Info("%s storage protocol stopped.", k)
		}
	}

	return nil
}

func (ss *StoreSvc) getSidDocFunc() func(versionId string) (*sid.SidDocument, error) {
	return func(versionId string) (*sid.SidDocument, error) {
		return ss.chainSvc.GetSidDocument(ss.ctx, versionId)
	}
}

func (ss *StoreSvc) getStorageProtocolAndPeer(
	ctx context.Context,
	targetAddress string,
) (StorageProtocol, string, error) {
	var sp StorageProtocol
	var err error
	peer := ""
	if targetAddress == ss.nodeAddress {
		sp = ss.storageProtocolMap["local"]
	} else {
		sp = ss.storageProtocolMap["stream"]
		peer, err = ss.chainSvc.GetNodePeer(ctx, targetAddress)
	}
	return sp, peer, err
}

func (ss *StoreSvc) updateShardError(shard types.ShardInfo, err error) {
	shard.LastErr = err.Error()
	err = utils.SaveShard(ss.ctx, ss.orderDs, shard)
	if err != nil {
		log.Warnf("put shard order=%d cid=%v error: %v", shard.OrderId, shard.Cid, err)
	}

}

func (ss *StoreSvc) ShardStatus(ctx context.Context, orderId uint64, cid cid.Cid) (types.ShardInfo, error) {
	return utils.GetShard(ctx, ss.orderDs, orderId, cid)
}

func (ss *StoreSvc) getPendingShardList(ctx context.Context) ([]types.ShardInfo, error) {
	shardKeys, err := ss.getShardKeyList(ctx)
	if err != nil {
		return nil, err
	}
	// TODO: optimize add a pending list in OrderShards
	var pending []types.ShardInfo
	for _, shardKey := range shardKeys {
		shard, err := utils.GetShard(ctx, ss.orderDs, shardKey.OrderId, shardKey.Cid)
		if err != nil {
			return nil, err
		}
		if shard.State != types.ShardStateComplete {
			pending = append(pending, shard)
		}
	}
	return pending, nil
}

func (ss *StoreSvc) getShardKeyList(ctx context.Context) ([]types.ShardKey, error) {
	index, err := utils.GetShardIndex(ctx, ss.orderDs)
	if err != nil {
		return nil, err
	}
	return index.All, nil
}

func (ss *StoreSvc) ShardList(ctx context.Context) ([]types.ShardInfo, error) {
	shardKeys, err := ss.getShardKeyList(ctx)
	if err != nil {
		return nil, err
	}

	var shardInfos []types.ShardInfo
	for _, shardKey := range shardKeys {
		shard, err := utils.GetShard(ctx, ss.orderDs, shardKey.OrderId, shardKey.Cid)
		if err != nil {
			return nil, err
		}
		shardInfos = append(shardInfos, shard)
	}
	return shardInfos, nil
}

func (ss *StoreSvc) ShardFix(ctx context.Context, orderId uint64, cid cid.Cid) error {
	shardInfo, err := utils.GetShard(ctx, ss.orderDs, orderId, cid)
	if err != nil {
		return nil
	}

	ss.taskChan <- shardInfo
	return nil
}
