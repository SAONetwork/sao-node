package gateway

import (
	"context"
	"fmt"
	"io"
	"sao-storage-node/node/chain"
	"sao-storage-node/node/utils"
	"sao-storage-node/store"
	"sao-storage-node/types"
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
	OrderId  uint64
	DataId   string
	CommitId string
	Cids     []string
}

type FetchResult struct {
	Cid     string
	Content []byte
}

type GatewaySvcApi interface {
	QueryMeta(ctx context.Context, key string) (*types.Model, error)
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

func (os *GatewaySvc) QueryMeta(ctx context.Context, key string) (*types.Model, error) {
	var res *modeltypes.QueryGetMetadataResponse = nil
	var err error
	var dataId string
	if utils.IsDataId(key) {
		dataId = key
	} else {
		dataId, err = os.chainSvc.QueryDataId(ctx, key)
		if err != nil {
			return nil, err
		}
	}
	res, err = os.chainSvc.QueryMeta(ctx, dataId)
	if err != nil {
		return nil, err
	}

	log.Debugf("QueryMeta succeed. meta=%v", res.Metadata)

	commitId := res.Metadata.DataId
	if len(res.Metadata.Commits) > 1 {
		commitId = res.Metadata.Commits[len(res.Metadata.Commits)-1]
	}

	return &types.Model{
		DataId:  res.Metadata.DataId,
		Alias:   res.Metadata.Alias,
		GroupId: res.Metadata.FamilyId,
		Creator: res.Metadata.Owner,
		OrderId: res.Metadata.OrderId,
		Tags:    res.Metadata.Tags,
		// Cid: N/a,
		ChunkCids: res.Metadata.Cids,
		Shards:    res.Shards,
		CommitId:  commitId,
		Commits:   res.Metadata.Commits,
		// Content: N/a,
		ExtendInfo: res.Metadata.ExtendInfo,
	}, nil
}

func (os *GatewaySvc) FetchContent(ctx context.Context, meta *types.Model) (*FetchResult, error) {
	contentList := make([][]byte, len(meta.Shards))
	for key, shard := range meta.Shards {
		shardCid, err := cid.Decode(shard.Cid)
		if err != nil {
			return nil, err
		}

		addr, err := os.chainSvc.GetNodePeer(ctx, key)
		if err != nil {
			return nil, err
		}

		var shardContent []byte
		if key == os.nodeAddress {
			// local shard
			if os.storeManager == nil {
				return nil, xerrors.Errorf("local store manager not found")
			}
			reader, err := os.storeManager.Get(ctx, shardCid)
			if err != nil {
				return nil, err
			}
			shardContent, err = io.ReadAll(reader)
			if err != nil {
				return nil, err
			}
		} else {
			// remote shard
			shardContent, err = os.shardStreamHandler.Fetch(addr, shardCid)
			if err != nil {
				return nil, err
			}
		}
		contentList[shard.Id] = shardContent
	}

	var content []byte
	for _, c := range contentList {
		content = append(content, c...)
	}

	return &FetchResult{
		Cid:     "123456",
		Content: content,
	}, nil
}

func (os *GatewaySvc) CommitModel(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*CommitResult, error) {
	// TODO: consider store node may ask earlier than file split
	// TODO: if big data, consider store to staging dir.
	// TODO: support split file.
	// TODO: support marshal any content
	err := StageShard(os.stagingPath, creator, orderMeta.Cid, content)
	if err != nil {
		return nil, err
	}

	commitId := utils.GenerateCommitId()
	if orderMeta.DataId == "" {
		orderMeta.DataId = commitId
	}
	orderMeta.CommitId = commitId

	if !orderMeta.TxSent {
		metadata := fmt.Sprintf(
			`{"alias": "%s", "dataId": "%s", "extenInfo": "%s", "familyId": "%s"}`,
			orderMeta.Alias,
			orderMeta.DataId,
			orderMeta.ExtenInfo,
			orderMeta.GroupId,
			// orderMeta.CommitId,
		)

		log.Info("metadata: ", metadata)

		orderId, txId, err := os.chainSvc.StoreOrder(ctx, os.nodeAddress, creator, os.nodeAddress, orderMeta.Cid, orderMeta.Duration, orderMeta.Replica, metadata)
		if err != nil {
			return nil, err
		}
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderId, txId)
		orderMeta.OrderId = orderId
		orderMeta.TxId = txId
		orderMeta.TxSent = true
	} else {
		txId, err := os.chainSvc.OrderReady(ctx, os.nodeAddress, orderMeta.OrderId)
		if err != nil {
			return nil, err
		}
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderMeta.OrderId, txId)

		orderMeta.TxId = txId
		orderMeta.TxSent = true
	}

	doneChan := make(chan chain.OrderCompleteResult)
	err = os.chainSvc.SubscribeOrderComplete(ctx, orderMeta.OrderId, doneChan)
	if err != nil {
		return nil, err
	}

	timeout := false
	select {
	case _ = <-doneChan:
		// dataId = result.DataId
	case <-time.After(chain.Blocktime * time.Duration(orderMeta.CompleteTimeoutBlocks)):
		timeout = true
	case <-ctx.Done():
		timeout = true
	}
	close(doneChan)

	err = os.chainSvc.UnsubscribeOrderComplete(ctx, orderMeta.OrderId)
	if err != nil {
		log.Error(err)
	} else {
		log.Debugf("UnsubscribeOrderComplete")
	}

	log.Debugf("unstage shard /%s/%v", creator, orderMeta.Cid)
	err = UnstageShard(os.stagingPath, creator, orderMeta.Cid)
	if err != nil {
		return nil, err
	}

	if timeout {
		// TODO: timeout handling
		return nil, errors.Errorf("process order %d timeout.", orderMeta.OrderId)
	} else {
		cids := make([]string, 1)
		cids[0] = orderMeta.Cid.String()
		log.Debugf("order %d complete: dataId=%s", orderMeta.OrderId, orderMeta.DataId)
		return &CommitResult{
			OrderId:  orderMeta.OrderId,
			DataId:   orderMeta.DataId,
			CommitId: orderMeta.CommitId,
			Cids:     cids,
		}, nil
	}
}

func (cs *GatewaySvc) Stop(ctx context.Context) error {
	log.Info("stopping order service...")
	cs.shardStreamHandler.Stop(ctx)

	return nil
}
