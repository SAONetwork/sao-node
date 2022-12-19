package gateway

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"
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
	Stop(ctx context.Context) error
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
}

func NewGatewaySvc(
	ctx context.Context,
	nodeAddress string,
	chainSvc *chain.ChainSvc,
	host host.Host,
	cfg *config.Node,
	storeManager *store.StoreManager,
	notifyChan map[string]chan interface{},
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

				shardContent, err = gs.shardStreamHandler.Fetch(req, peerInfo, shardCid)
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

func (gs *GatewaySvc) CommitModel(ctx context.Context, clientProposal *types.OrderStoreProposal, orderId uint64, content []byte) (*CommitResult, error) {
	// TODO: consider store node may ask earlier than file split
	// TODO: if big data, consider store to staging dir.
	// TODO: support split file.
	// TODO: support marshal any content
	orderProposal := clientProposal.Proposal
	err := StageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid, content)
	if err != nil {
		return nil, err
	}

	var txHash string
	var shards map[string]*saotypes.ShardMeta
	var txType types.AssignTxType
	var height int64
	if orderId == 0 {
		resp, txId, h, err := gs.chainSvc.StoreOrder(ctx, gs.nodeAddress, clientProposal)
		if err != nil {
			return nil, err
		}
		orderId = resp.OrderId
		shards = resp.Shards
		txHash = txId
		txType = types.AssignTxTypeStore
		height = h
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderId, txId)
		log.Infof("StoreOrder tx succeed. shards=%v", resp.Shards)
	} else {
		log.Debugf("Sending OrderReady... orderId=%d", orderId)
		resp, txId, h, err := gs.chainSvc.OrderReady(ctx, gs.nodeAddress, orderId)
		if err != nil {
			return nil, err
		}
		orderId = resp.OrderId
		shards = resp.Shards
		txHash = txId
		height = h
		txType = types.AssignTxTypeReady
		log.Infof("OrderReady tx succeed. orderId=%d tx=%s", orderId, txId)
		log.Infof("OrderReady tx succeed. shards=%v", resp.Shards)
	}

	log.Infof("assigning shards to nodes...")
	// assign shards to storage nodes

	for node, shard := range shards {
		var shardAssignReq = types.ShardAssignReq{
			OrderId:      orderId,
			TxHash:       txHash,
			Assignee:     node,
			Height:       height,
			AssignTxType: txType,
		}
		if node == gs.nodeAddress {
			if c, exists := gs.notifyChan[types.ShardAssignProtocol]; exists {
				c <- shardAssignReq
			} else {
				log.Errorf("assign order %d shards to node %s failed: %v", orderId, node, "no channel defined")
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
				log.Errorf("assign order %d shards to node %s failed: %v", orderId, node, err)
				continue
			}
			if resp.Code == 0 {
				log.Infof("assigned order %d shard to node %s.", orderId, node)
			} else {
				log.Errorf("assigned order %d shards to node %s failed: %v", orderId, node, resp.Message)
			}
		}
	}

	// TODO: wsevent
	//doneChan := make(chan chain.OrderCompleteResult)
	//err = gs.chainSvc.SubscribeOrderComplete(ctx, orderId, doneChan)
	//if err != nil {
	//	return nil, err
	//}

	log.Infof("waiting for all nodes order %d shard completion.", orderId)
	gs.completeChan[orderId] = make(chan struct{})

	timeout := false
	select {
	case <-gs.completeChan[orderId]:
		log.Debugf("complete channel done. order %d completes", orderId)
	case <-time.After(chain.Blocktime * time.Duration(clientProposal.Proposal.Timeout)):
		timeout = true
	case <-ctx.Done():
		timeout = true
	}
	close(gs.completeChan[orderId])
	delete(gs.completeChan, orderId)

	// TODO: wsevent
	//err = gs.chainSvc.UnsubscribeOrderComplete(ctx, orderId)
	//if err != nil {
	//	log.Error(err)
	//} else {
	//	log.Debugf("UnsubscribeOrderComplete")
	//}

	log.Debugf("unstage shard %s/%s/%v", gs.stagingPath, orderProposal.Owner, orderProposal.Cid)
	err = UnstageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid)
	if err != nil {
		return nil, err
	}

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

func (cs *GatewaySvc) Stop(ctx context.Context) error {
	log.Info("stopping order service...")
	cs.shardStreamHandler.Stop(ctx)

	return nil
}
