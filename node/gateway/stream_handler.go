package gateway

import (
	"context"
	"fmt"
	"sao-node/chain"
	"sao-node/node/transport"
	"sao-node/types"
	"sync"
	"time"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type ShardStreamHandler struct {
	ctx          context.Context
	host         host.Host
	stagingPath  string
	chainSvc     chain.ChainSvcApi
	nodeAddress  string
	completeChan map[uint64]chan struct{}
}

var (
	handler *ShardStreamHandler
	once    sync.Once
)

func NewShardStreamHandler(ctx context.Context, host host.Host, path string, chainSvc chain.ChainSvcApi, nodeAddress string, completeChan map[uint64]chan struct{}) *ShardStreamHandler {
	once.Do(func() {
		handler = &ShardStreamHandler{
			ctx:          ctx,
			host:         host,
			stagingPath:  path,
			chainSvc:     chainSvc,
			nodeAddress:  nodeAddress,
			completeChan: completeChan,
		}

		host.SetStreamHandler(types.ShardStoreProtocol, handler.HandleShardStream)
		host.SetStreamHandler(types.ShardCompleteProtocol, handler.HandleShardCompleteStream)
	})

	return handler
}

func (ssh *ShardStreamHandler) HandleShardCompleteStream(s network.Stream) {
	defer s.Close()

	respond := func(resp types.ShardCompleteResp) {
		err := resp.Marshal(s, "json")
		if err != nil {
			log.Error(types.Wrap(types.ErrMarshalFailed, err))
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(types.Wrap(types.ErrCloseFileFailed, err))
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardCompleteReq
	err := req.Unmarshal(s, "json")
	if err != nil {
		log.Error(types.Wrap(types.ErrUnMarshalFailed, err))
		respond(types.ShardCompleteResp{
			Code:    types.ErrorCodeInvalidRequest,
			Message: fmt.Sprintf("failed to unmarshal request: %v", err),
		})
		return
	}
	log.Debugf("receive ShardCompleteReq")

	if req.Code != 0 {
		// TODO: notify channel that storage node can't handle this shard.
		log.Debugf("storage node can't handle order %d shards %v: %s", req.OrderId, req.Cids, req.Message)
		respond(types.ShardCompleteResp{Code: 0})
		return
	}
	// query tx
	resultTx, err := ssh.chainSvc.GetTx(ssh.ctx, req.TxHash, req.Height)
	if err != nil {
		respond(types.ShardCompleteResp{
			Code:    types.ErrorCodeInternalErr,
			Message: fmt.Sprintf("internal error: %v", err),
		})
		return
	}
	if resultTx.TxResult.Code == 0 {
		txb := tx.Tx{}
		err = txb.Unmarshal(resultTx.Tx)
		if err != nil {
			respond(types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidTx,
				Message: fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			})
			return
		}

		m := saotypes.MsgComplete{}
		err = m.Unmarshal(txb.Body.Messages[0].Value)
		if err != nil {
			respond(types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidTx,
				Message: fmt.Sprintf("tx %s body is invalid.", resultTx.Tx),
			})
			return
		}

		order, err := ssh.chainSvc.GetOrder(ssh.ctx, m.OrderId)
		if err != nil {
			respond(types.ShardCompleteResp{
				Code:    types.ErrorCodeInternalErr,
				Message: fmt.Sprintf("internal error: %v", err),
			})
			return
		}

		if order.Provider != ssh.nodeAddress {
			respond(types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidOrderProvider,
				Message: fmt.Sprintf("order %d provider is %s, not %s", m.OrderId, order.Provider, ssh.nodeAddress),
			})
			return
		}

		shardCids := make(map[string]struct{})
		for key, shard := range order.Shards {
			if key == m.Creator {
				shardCids[shard.Cid] = struct{}{}
			}
		}
		if len(shardCids) <= 0 {
			respond(types.ShardCompleteResp{
				Code:    types.ErrorCodeInvalidProvider,
				Message: fmt.Sprintf("order %d doesn't have shard provider %s", m.OrderId, m.Creator),
			})
			return
		}

		for _, cid := range req.Cids {
			if _, ok := shardCids[cid.String()]; !ok {
				respond(types.ShardCompleteResp{
					Code:    types.ErrorCodeInvalidShardCid,
					Message: fmt.Sprintf("%v is not in the given order %d", cid.String(), m.OrderId),
				})
				return
			}
		}

		if order.Status == saotypes.OrderCompleted {
			// update channel.
			ssh.completeChan[m.OrderId] <- struct{}{}
		}
		respond(types.ShardCompleteResp{Code: 0})
	} else {
		// respond storage node to re-handle this shard.
		respond(types.ShardCompleteResp{
			Code:    types.ErrorCodeInvalidTx,
			Message: fmt.Sprintf("tx %s failed with code %d", req.TxHash, resultTx.TxResult.Code),
		})
	}
}

func (ssh *ShardStreamHandler) HandleShardStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardReq
	err := req.Unmarshal(s, "json")
	if err != nil {
		log.Error(types.Wrap(types.ErrUnMarshalFailed, err))
		// TODO: respond error
		return
	}
	log.Debugf("receive ShardReq: orderId=%d cid=%v", req.OrderId, req.Cid, req.RequestId)

	contentBytes, err := GetStagedShard(ssh.stagingPath, req.Owner, req.Cid)
	if err != nil {
		log.Error(err)
		// TODO: respond error
		return
	}
	var resp = &types.ShardResp{
		OrderId:    req.OrderId,
		Cid:        req.Cid,
		Content:    contentBytes,
		RequestId:  req.RequestId,
		ResponseId: time.Now().UnixMilli(),
	}
	log.Debugf("send ShardResp(requestId=%d,responseId=%d): len(Content)=%d", req.RequestId, resp.ResponseId, len(contentBytes))

	err = resp.Marshal(s, "json")
	if err != nil {
		log.Error(types.Wrap(types.ErrMarshalFailed, err))
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(types.Wrap(types.ErrCloseFileFailed, err))
		return
	}
}

func (ssh *ShardStreamHandler) Fetch(req *types.MetadataProposal, addr string, orderId uint64, shardCid cid.Cid) ([]byte, error) {
	a, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, types.Wrap(types.ErrInvalidServerAddress, err)
	}
	pi, err := peer.AddrInfoFromP2pAddr(a)
	if err != nil {
		return nil, types.Wrap(types.ErrInvalidServerAddress, err)
	}
	err = ssh.host.Connect(ssh.ctx, *pi)
	if err != nil {
		return nil, types.Wrap(types.ErrConnectFailed, err)
	}
	stream, err := ssh.host.NewStream(ssh.ctx, pi.ID, types.ShardLoadProtocol)
	if err != nil {
		return nil, types.Wrap(types.ErrCreateStreamFailed, err)
	}
	defer stream.Close()
	log.Infof("open stream(%s) to storage node %s", types.ShardLoadProtocol, addr)

	// Set a deadline on reading from the stream so it doesn't hang
	_ = stream.SetReadDeadline(time.Now().Add(300 * time.Second))
	defer stream.SetReadDeadline(time.Time{}) // nolint

	request := types.ShardReq{
		Cid:       shardCid,
		OrderId:   orderId,
		Proposal:  req,
		RequestId: time.Now().UnixMilli(),
	}
	log.Infof("send ShardReq with cid:%v, to the storage node %s", request.Cid, addr)

	var resp types.ShardResp
	for retryTimes := 0; ; retryTimes++ {
		if err = transport.DoRequest(ssh.ctx, stream, &request, &resp, "json"); err != nil {
			if retryTimes > 2 {
				return nil, err
			} else {
				log.Error(err)
			}
			time.Sleep(time.Second * 10)
		} else {
			break
		}
	}

	log.Debugf("receive ShardResp(requestId=%d,responseId=%d) with content length:%d, from the storage node %s", resp.RequestId, resp.ResponseId, len(resp.Content), addr)

	return resp.Content, nil
}

func (ssh *ShardStreamHandler) Stop(ctx context.Context) error {
	log.Info("stopping shard stream handler...")
	ssh.host.RemoveStreamHandler(types.ShardStoreProtocol)
	ssh.host.RemoveStreamHandler(types.ShardCompleteProtocol)
	return nil
}
