package order

import (
	"context"
	"sao-storage-node/node/chain"
	"sao-storage-node/types"
	"time"

	logging "github.com/ipfs/go-log/v2"
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

type QueryResult struct {
	OrderId  uint64
	DataId   string
	Alias    string
	Tags     string
	CommitId string
	Content  []byte
	Cids     []string
	Type     types.ModelType
}

type OrderSvcApi interface {
	Commit(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*CommitResult, error)
	Query(ctx context.Context, key string) (*QueryResult, error)
	// Stop(ctx context.Context) error
}

type OrderSvc struct {
	ctx         context.Context
	chainSvc    *chain.ChainSvc
	nodeAddress string
	stagingPath string
}

func NewOrderSvc(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, stagingPath string) *OrderSvc {
	cs := &OrderSvc{
		ctx:         ctx,
		chainSvc:    chainSvc,
		nodeAddress: nodeAddress,
		stagingPath: stagingPath,
	}

	return cs
}

func (cs *OrderSvc) Commit(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*CommitResult, error) {
	// TODO: consider store node may ask earlier than file split
	// TODO: if big data, consider store to staging dir.
	// TODO: support split file.
	// TODO: support marshal any content
	log.Infof("stage shard /%s/%v", creator, orderMeta.Cid)
	err := StageShard(cs.stagingPath, creator, orderMeta.Cid, content)
	if err != nil {
		return nil, err
	}

	if !orderMeta.TxSent {
		orderId, txId, err := cs.chainSvc.StoreOrder(ctx, cs.nodeAddress, creator, cs.nodeAddress, orderMeta.Cid, orderMeta.Duration, orderMeta.Replica)
		if err != nil {
			return nil, err
		}
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderId, txId)
		orderMeta.OrderId = orderId
		orderMeta.TxId = txId
		orderMeta.TxSent = true
	} else {
		txId, err := cs.chainSvc.OrderReady(ctx, cs.nodeAddress, orderMeta.OrderId)
		if err != nil {
			return nil, err
		}
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderMeta.OrderId, txId)

		orderMeta.TxId = txId
		orderMeta.TxSent = true
	}

	log.Infof("start SubscribeOrderComplete")
	doneChan := make(chan chain.OrderCompleteResult)
	err = cs.chainSvc.SubscribeOrderComplete(ctx, orderMeta.OrderId, doneChan)
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

	err = cs.chainSvc.UnsubscribeOrderComplete(ctx, orderMeta.OrderId)
	if err != nil {
		log.Error(err)
	} else {
		log.Info("UnsubscribeOrderComplete")
	}

	log.Infof("unstage shard /%s/%v", creator, orderMeta.Cid)
	err = UnstageShard(cs.stagingPath, creator, orderMeta.Cid)
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

func (cs *OrderSvc) Query(ctx context.Context, key string) (*QueryResult, error) {
	return nil, xerrors.Errorf("not implemented yet")
}
