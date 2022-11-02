package order

import (
	"context"
	"io"
	"sao-storage-node/node/chain"
	"sao-storage-node/store"
	"sao-storage-node/types"
	"strconv"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
	"golang.org/x/xerrors"
)

var log = logging.Logger("order")

type CommitResult struct {
	OrderId  uint64
	DataId   string
	CommitId string
	Cids     []string
}

type FetchResult struct {
	DataId  string
	Alias   string
	Cid     string
	Content []byte
}

type OrderSvcApi interface {
	Commit(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*CommitResult, error)
	Query(ctx context.Context, key string) (*types.OrderMeta, error)
	Fetch(ctx context.Context, orderId uint64) (*FetchResult, error)
	Stop(ctx context.Context) error
}

type OrderSvc struct {
	ctx                context.Context
	chainSvc           *chain.ChainSvc
	shardStreamHandler *ShardStreamHandler
	storeManager       *store.StoreManager
	nodeAddress        string
	stagingPath        string
}

func NewOrderSvc(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, host host.Host, stagingPath string, storeManager *store.StoreManager) *OrderSvc {
	cs := &OrderSvc{
		ctx:                ctx,
		chainSvc:           chainSvc,
		shardStreamHandler: NewShardStreamHandler(ctx, host, stagingPath),
		storeManager:       storeManager,
		nodeAddress:        nodeAddress,
		stagingPath:        stagingPath,
	}

	return cs
}

func (os *OrderSvc) Commit(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*CommitResult, error) {
	// TODO: consider store node may ask earlier than file split
	// TODO: if big data, consider store to staging dir.
	// TODO: support split file.
	// TODO: support marshal any content
	log.Infof("stage shard /%s/%v", creator, orderMeta.Cid)
	err := StageShard(os.stagingPath, creator, orderMeta.Cid, content)
	if err != nil {
		return nil, err
	}

	if !orderMeta.TxSent {
		orderId, txId, err := os.chainSvc.StoreOrder(ctx, os.nodeAddress, creator, os.nodeAddress, orderMeta.Cid, orderMeta.Duration, orderMeta.Replica)
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

	log.Infof("start SubscribeOrderComplete")
	doneChan := make(chan chain.OrderCompleteResult)
	err = os.chainSvc.SubscribeOrderComplete(ctx, orderMeta.OrderId, doneChan)
	if err != nil {
		return nil, err
	}

	dataId := ""
	timeout := false
	select {
	case result := <-doneChan:
		dataId = result.DataId
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
		log.Info("UnsubscribeOrderComplete")
	}

	log.Infof("unstage shard /%s/%v", creator, orderMeta.Cid)
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
		log.Infof("order %d complete: dataId=%s", orderMeta.OrderId, dataId)
		return &CommitResult{
			OrderId: orderMeta.OrderId,
			DataId:  dataId,
			Cids:    cids,
		}, nil
	}
}

func (os *OrderSvc) Query(ctx context.Context, key string) (*types.OrderMeta, error) {
	var cids = make([]string, 1)
	fakeCid := "bafkreih36a72gdu7dwozyaegev47bscepunrlt64lcoyxcejuhnlnjnw6e"
	cids[0] = fakeCid

	uintNum, _ := strconv.Atoi(key)
	orderId := uint64(uintNum)

	return &types.OrderMeta{
		DataId:    key,
		Alias:     key,
		CommitId:  key,
		OrderId:   orderId,
		ChunkCids: cids,
	}, nil
}

func (os *OrderSvc) Fetch(ctx context.Context, orderId uint64) (*FetchResult, error) {
	order, err := os.chainSvc.GetOrder(ctx, orderId)
	if err != nil {
		return nil, err
	}

	contentList := make([][]byte, len(order.Shards))
	for key, shard := range order.Shards {
		log.Info("shard: ", shard)
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
	log.Info("contentList: ", contentList)

	var content []byte
	for _, c := range contentList {
		content = append(content, c...)
	}

	log.Info("content: ", content)

	return &FetchResult{
		DataId:  "12234",
		Alias:   "fasdfasdfasdf",
		Cid:     "123456",
		Content: content,
	}, nil
}

func (cs *OrderSvc) Stop(ctx context.Context) error {
	log.Info("stopping order service...")
	cs.shardStreamHandler.Stop(ctx)

	return nil
}
