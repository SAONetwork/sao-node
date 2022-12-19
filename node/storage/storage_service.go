package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sao-node/chain"
	"sao-node/node/transport"
	"sao-node/store"
	"sao-node/types"
	"strings"
	"time"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"

	sid "github.com/SaoNetwork/sao-did/sid"
	logging "github.com/ipfs/go-log/v2"

	saodid "github.com/SaoNetwork/sao-did"
	saodidtypes "github.com/SaoNetwork/sao-did/types"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var log = logging.Logger("storage")

type StoreSvc struct {
	nodeAddress  string
	chainSvc     *chain.ChainSvc
	taskChan     chan *chain.ShardTask
	host         host.Host
	stagingPath  string
	storeManager *store.StoreManager
	ctx          context.Context
}

func NewStoreService(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, host host.Host, stagingPath string, storeManager *store.StoreManager) (*StoreSvc, error) {
	ss := StoreSvc{
		nodeAddress:  nodeAddress,
		chainSvc:     chainSvc,
		taskChan:     make(chan *chain.ShardTask),
		host:         host,
		stagingPath:  stagingPath,
		storeManager: storeManager,
		ctx:          ctx,
	}

	host.SetStreamHandler(types.ShardLoadProtocol, ss.HandleShardStream)
	host.SetStreamHandler(types.ShardAssignProtocol, ss.HandleShardAssignStream)

	// TODO: wsevent
	//if err := ss.chainSvc.SubscribeShardTask(ctx, ss.nodeAddress, ss.taskChan); err != nil {
	//	return nil, err
	//}

	return &ss, nil
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

func (ss *StoreSvc) process(ctx context.Context, task *chain.ShardTask) error {
	log.Debugf("processing task: order id=%d gateway=%s shard_cid=%v", task.OrderId, task.Gateway, task.Cid)
	log.Debugf("processing task: order operation=%s shard operation=%s", task.OrderOperation, task.ShardOperation)

	var shard []byte
	var err error

	// check if it's a renew order(Operation is 3)
	if task.OrderOperation != "3" || task.ShardOperation != "3" {
		// check if gateway is node itself
		if task.Gateway == ss.nodeAddress {
			shard, err = ss.getShardFromLocal(task.Owner, task.Cid)
			if err != nil {
				log.Warn("skip the known error: ", err.Error())
				return err
			}
		} else {
			shard, err = ss.getShardFromGateway(ctx, task.Owner, task.Gateway, task.OrderId, task.Cid)
			if err != nil {
				return err
			}
		}

		// store to backends
		_, err = ss.storeManager.Store(ctx, task.Cid, bytes.NewReader(shard))
		if err != nil {
			return err
		}
	} else {
		// make sure the data is still there
		isExist := ss.storeManager.IsExist(ctx, task.Cid)
		if !isExist {
			return xerrors.Errorf("shard with cid %s not found", task.Cid)
		}
	}

	log.Info("Complete order")
	txHash, err := ss.chainSvc.CompleteOrder(ctx, ss.nodeAddress, task.OrderId, task.Cid, int32(len(shard)))
	if err != nil {
		return err
	}
	log.Infof("Complete order succeed: txHash: %s, OrderId: %d, cid: %s", txHash, task.OrderId, task.Cid)

	peerInfos, err := ss.chainSvc.GetNodePeer(ctx, task.Gateway)
	if err != nil {
		return err
	}
	resp := types.ShardCompleteResp{}
	err = transport.HandleRequest(ctx, peerInfos, ss.host, types.ShardCompleteProtocol, &types.ShardCompleteReq{
		OrderId: task.OrderId,
		Cids:    []cid.Cid{task.Cid},
		TxHash:  txHash,
		Code:    0,
	}, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (ss *StoreSvc) getShardFromLocal(creator string, cid cid.Cid) ([]byte, error) {
	path, err := homedir.Expand(ss.stagingPath)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%v", cid)
	bytes, err := os.ReadFile(filepath.Join(path, creator, filename))
	if err != nil {
		return nil, err
	} else {
		return bytes, nil
	}
}

func (ss *StoreSvc) getShardFromGateway(ctx context.Context, owner string, gateway string, orderId uint64, cid cid.Cid) ([]byte, error) {
	peerInfos, err := ss.chainSvc.GetNodePeer(ctx, gateway)
	if err != nil {
		return nil, err
	}
	resp := types.ShardResp{}
	err = transport.HandleRequest(ctx, peerInfos, ss.host, types.ShardStoreProtocol, &types.ShardReq{
		Owner:   owner,
		OrderId: orderId,
		Cid:     cid,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Content, nil

}

func (ss *StoreSvc) Stop(ctx context.Context) error {
	// TODO: wsevent
	//if err := ss.chainSvc.UnsubscribeShardTask(ctx, ss.nodeAddress); err != nil {
	//	return err
	//}
	close(ss.taskChan)
	return nil
}

func (ss *StoreSvc) getSidDocFunc() func(versionId string) (*sid.SidDocument, error) {
	return func(versionId string) (*sid.SidDocument, error) {
		return ss.chainSvc.GetSidDocument(ss.ctx, versionId)
	}
}

func (ss *StoreSvc) HandleShardAssignStream(s network.Stream) {
	defer s.Close()

	respond := func(resp types.ShardAssignResp) {
		err := resp.Marshal(s, "json")
		if err != nil {
			log.Error(err.Error())
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(err.Error())
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardAssignReq
	err := req.Unmarshal(s, "json")
	if err != nil {
		respond(types.ShardAssignResp{
			Code:    types.ErrorCodeInvalidRequest,
			Message: fmt.Sprintf("failed to unmarshal request: %v", err),
		})
		return
	}

	// validate request
	if req.Assignee != ss.nodeAddress {
		respond(types.ShardAssignResp{
			Code:    types.ErrorCodeInvalidShardAssignee,
			Message: fmt.Sprintf("shard assignee is %s, but current node is %s", req.Assignee, ss.nodeAddress),
		})
		return
	}

	resultTx, err := ss.chainSvc.GetTx(ss.ctx, req.TxHash)
	if resultTx.TxResult.Code == 0 {
		txb := tx.Tx{}
		err = txb.Unmarshal(resultTx.Tx)
		if err != nil {
			respond(types.ShardAssignResp{
				Code:    types.ErrorCodeInvalidTx,
				Message: fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			})
			return
		}

		if req.AssignTxType == types.AssignTxTypeStore {
			m := saotypes.MsgStore{}
			err = m.Unmarshal(txb.Body.Messages[0].Value)
		} else {
			m := saotypes.MsgReady{}
			err = m.Unmarshal(txb.Body.Messages[0].Value)
		}
		if err != nil {
			respond(types.ShardAssignResp{
				Code:    types.ErrorCodeInvalidTx,
				Message: fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			})
			return
		}

		order, err := ss.chainSvc.GetOrder(ss.ctx, req.OrderId)
		if err != nil {
			respond(types.ShardAssignResp{
				Code:    types.ErrorCodeInternalErr,
				Message: fmt.Sprintf("internal error: %v", err),
			})
			return
		}

		var shardCids []string
		for key, shard := range order.Shards {
			if key == ss.nodeAddress {
				shardCids = append(shardCids, shard.Cid)
			}
		}
		if len(shardCids) <= 0 {
			respond(types.ShardAssignResp{
				Code:    types.ErrorCodeInvalidProvider,
				Message: fmt.Sprintf("order %d doesn't have shard provider %s", req.OrderId, ss.nodeAddress),
			})
			return
		}
		var shardTasks []*chain.ShardTask
		for _, shardCid := range shardCids {
			cid, err := cid.Decode(shardCid)
			if err != nil {
				respond(types.ShardAssignResp{
					Code:    types.ErrorCodeInvalidShardCid,
					Message: fmt.Sprintf("invalid cid %s", shardCid),
				})
				return
			}

			shardTasks = append(shardTasks, &chain.ShardTask{
				Owner:          order.Owner,
				OrderId:        req.OrderId,
				Gateway:        order.Provider,
				Cid:            cid,
				OrderOperation: "",
				ShardOperation: "",
			})
		}
		for _, task := range shardTasks {
			ss.taskChan <- task
		}

		respond(types.ShardAssignResp{Code: 0})
		return
	} else {
		respond(types.ShardAssignResp{
			Code:    types.ErrorCodeInvalidTx,
			Message: fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
		})
		return
	}
}

func (ss *StoreSvc) HandleShardStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardReq
	err := req.Unmarshal(s, "json")
	if err != nil {
		log.Error(err)
		return
	}
	log.Debugf("receive ShardReq: orderId=%d cid=%v", req.OrderId, req.Cid)

	didManager, err := saodid.NewDidManagerWithDid(req.Proposal.Proposal.Owner, ss.getSidDocFunc())
	if err != nil {
		log.Error(err)
		return
	}

	proposalBytes, err := req.Proposal.Proposal.Marshal()
	if err != nil {
		log.Error(err)
		return
	}

	_, err = didManager.VerifyJWS(saodidtypes.GeneralJWS{
		Payload: base64url.Encode(proposalBytes),
		Signatures: []saodidtypes.JwsSignature{
			saodidtypes.JwsSignature(req.Proposal.JwsSignature),
		},
	})

	if err != nil {
		log.Errorf("verify client order proposal signature failed: %v", err)
		return
	}

	lastHeight, err := ss.chainSvc.GetLastHeight(ss.ctx)
	if err != nil {
		log.Error(err)
		return
	}
	peerInfo := string(s.Conn().RemotePeer())
	if strings.Contains(req.Proposal.Proposal.Gateway, peerInfo) {
		log.Errorf("invalid query, unexpect gateway:%s, should be %s", peerInfo, req.Proposal.Proposal.Gateway)
		return
	}
	if req.Proposal.Proposal.LastValidHeight < uint64(lastHeight) {
		log.Errorf("invalid query, LastValidHeight:%d > now:%d", req.Proposal.Proposal.LastValidHeight, lastHeight)
		return
	}

	reader, err := ss.storeManager.Get(ss.ctx, req.Cid)
	if err != nil {
		log.Error(err)
		return
	}
	shardContent, err := io.ReadAll(reader)
	if err != nil {
		log.Error(err)
		return
	}

	var resp = &types.ShardResp{
		OrderId: req.OrderId,
		Cid:     req.Cid,
		Content: shardContent,
	}
	log.Debugf("send ShardResp: Content len %d", len(shardContent))

	err = resp.Marshal(s, "json")
	if err != nil {
		log.Error(err.Error())
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}
}
