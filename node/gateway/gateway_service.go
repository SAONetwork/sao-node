package gateway

import (
	"context"
	"fmt"
	"io"
	"sao-storage-node/node/chain"
	"sao-storage-node/store"
	"sao-storage-node/types"
	"sao-storage-node/utils"
	"strings"
	"time"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	"golang.org/x/xerrors"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
)

var log = logging.Logger("order")

type CommitResult struct {
	OrderId uint64
	DataId  string
	Commit  string
	Commits []string
	Cid     string
	Shards  map[string]*modeltypes.ShardMeta
}

type FetchResult struct {
	Cid     string
	Content []byte
}

type GatewaySvcApi interface {
	QueryMeta(ctx context.Context, account string, key string, group string, height int64) (*types.Model, error)
	CommitModel(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*CommitResult, error)
	FetchContent(ctx context.Context, meta *types.Model) (*FetchResult, error)
	Stop(ctx context.Context) error
}

type GatewaySvc struct {
	ctx                context.Context
	chainSvc           *chain.ChainSvc
	shardStreamHandler *ShardStreamHandler
	storeManager       *store.StoreManager
	nodeAddress        string
	stagingPath        string
}

func NewGatewaySvc(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, host host.Host, stagingPath string, storeManager *store.StoreManager) *GatewaySvc {
	cs := &GatewaySvc{
		ctx:                ctx,
		chainSvc:           chainSvc,
		shardStreamHandler: NewShardStreamHandler(ctx, host, stagingPath),
		storeManager:       storeManager,
		nodeAddress:        nodeAddress,
		stagingPath:        stagingPath,
	}

	return cs
}

func (gs *GatewaySvc) QueryMeta(ctx context.Context, account string, key string, group string, height int64) (*types.Model, error) {
	var res *modeltypes.QueryGetMetadataResponse = nil
	var err error
	var dataId string
	if utils.IsDataId(key) {
		dataId = key
	} else {
		dataId, err = gs.chainSvc.QueryDataId(ctx, fmt.Sprintf("%s-%s-%s", account, key, group))
		if err != nil {
			return nil, err
		}
	}
	res, err = gs.chainSvc.QueryMeta(ctx, dataId, height)
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
		DataId:  res.Metadata.DataId,
		Alias:   res.Metadata.Alias,
		GroupId: res.Metadata.GroupId,
		Owner:   res.Metadata.Owner,
		OrderId: res.Metadata.OrderId,
		Tags:    res.Metadata.Tags,
		// Cid: res.Metadata.Cid,
		Shards:   res.Shards,
		CommitId: commitInfo[0],
		Commits:  res.Metadata.Commits,
		// Content: N/a,
		ExtendInfo: res.Metadata.ExtendInfo,
	}, nil
}

func (gs *GatewaySvc) FetchContent(ctx context.Context, meta *types.Model) (*FetchResult, error) {
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
			shardContent, err = gs.shardStreamHandler.Fetch(shard.Peer, shardCid)
			if err != nil {
				return nil, err
			}
		}
		contentList[shard.ShardId] = shardContent
	}

	var content []byte
	for _, c := range contentList {
		content = append(content, c...)
	}

	contentCid, err := utils.CaculateCid(content)
	if err != nil {
		return nil, err
	}
	if contentCid.String() != meta.Cid {
		log.Errorf("cid mismatch, expected %s, but got %s", meta.Cid, contentCid.String())
	}

	return &FetchResult{
		Cid:     contentCid.String(),
		Content: content,
	}, nil
}

func (gs *GatewaySvc) CommitModel(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*CommitResult, error) {
	// TODO: consider store node may ask earlier than file split
	// TODO: if big data, consider store to staging dir.
	// TODO: support split file.
	// TODO: support marshal any content
	err := StageShard(gs.stagingPath, creator, orderMeta.Cid, content)
	if err != nil {
		return nil, err
	}

	commitId := utils.GenerateCommitId()
	if orderMeta.DataId == "" {
		orderMeta.DataId = commitId
	}
	if orderMeta.CommitId == "" {
		orderMeta.CommitId = commitId
	}

	if !orderMeta.TxSent {
		var metadata string
		if orderMeta.IsUpdate {
			metadata = fmt.Sprintf(
				`{"dataId": "%s", "commit": "%s", "update": true}`,
				orderMeta.DataId,
				orderMeta.CommitId,
			)
		} else {
			metadata = fmt.Sprintf(
				`{"alias": "%s", "dataId": "%s", "extendInfo": "%s", "groupId": "%s", "commit": "%s", "update": false}`,
				orderMeta.Alias,
				orderMeta.DataId,
				orderMeta.ExtendInfo,
				orderMeta.GroupId,
				orderMeta.CommitId,
			)
		}

		orderId, txId, err := gs.chainSvc.StoreOrder(ctx, gs.nodeAddress, creator, gs.nodeAddress, orderMeta.Cid, orderMeta.Duration, orderMeta.Replica, metadata)
		if err != nil {
			return nil, err
		}
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderId, txId)
		orderMeta.OrderId = orderId
		orderMeta.TxId = txId
		orderMeta.TxSent = true
	} else {
		txId, err := gs.chainSvc.OrderReady(ctx, gs.nodeAddress, orderMeta.OrderId)
		if err != nil {
			return nil, err
		}
		log.Infof("OrderReady tx succeed. orderId=%d tx=%s", orderMeta.OrderId, txId)

		orderMeta.TxId = txId
		orderMeta.TxSent = true
	}

	doneChan := make(chan chain.OrderCompleteResult)
	err = gs.chainSvc.SubscribeOrderComplete(ctx, orderMeta.OrderId, doneChan)
	if err != nil {
		return nil, err
	}

	log.Debugf("SubscribeOrderComplete")

	timeout := false
	select {
	case <-doneChan:
	case <-time.After(chain.Blocktime * time.Duration(orderMeta.CompleteTimeoutBlocks)):
		timeout = true
	case <-ctx.Done():
		timeout = true
	}
	close(doneChan)

	err = gs.chainSvc.UnsubscribeOrderComplete(ctx, orderMeta.OrderId)
	if err != nil {
		log.Error(err)
	} else {
		log.Debugf("UnsubscribeOrderComplete")
	}

	log.Debugf("unstage shard %s/%s/%v", gs.stagingPath, creator, orderMeta.Cid)
	err = UnstageShard(gs.stagingPath, creator, orderMeta.Cid)
	if err != nil {
		return nil, err
	}

	if timeout {
		// TODO: timeout handling
		return nil, errors.Errorf("process order %d timeout.", orderMeta.OrderId)
	} else {
		meta, err := gs.chainSvc.QueryMeta(ctx, orderMeta.DataId, 0)
		if err != nil {
			return nil, err
		}
		log.Debugf("order %d complete: dataId=%s", meta.Metadata.OrderId, &meta.Metadata.DataId)

		return &CommitResult{
			OrderId: meta.Metadata.OrderId,
			DataId:  meta.Metadata.DataId,
			Commit:  meta.Metadata.Commit,
			Commits: meta.Metadata.Commits,
			Shards:  meta.Shards,
			Cid:     orderMeta.Cid.String(),
		}, nil
	}
}

func (cs *GatewaySvc) Stop(ctx context.Context) error {
	log.Info("stopping order service...")
	cs.shardStreamHandler.Stop(ctx)

	return nil
}
