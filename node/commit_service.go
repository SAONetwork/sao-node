package node

import (
	"context"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/tendermint/tendermint/rpc/client/http"
	"sao-storage-node/node/chain"
	"sao-storage-node/types"
	"time"
)

const subscriber_gateway = "gatewaynode"

type CommitResult struct {
	OrderId  uint64
	DataId   string
	CommitId string
}

type CommitSvc struct {
	ctx           context.Context
	chainSvc      *chain.ChainSvc
	nodeAddress   string
	chainListener *http.HTTP
	db            datastore.Batching
	host          host.Host
}

func NewCommitSvc(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, http *http.HTTP, db datastore.Batching, host host.Host) *CommitSvc {
	return &CommitSvc{
		ctx:           ctx,
		chainSvc:      chainSvc,
		nodeAddress:   nodeAddress,
		chainListener: http,
		db:            db,
		host:          host,
	}
}

func (cs *CommitSvc) Start() {
	log.Info("start commit service")
	cs.host.SetStreamHandler(ShardStoreProtocol, cs.handleShardStore)
}

func (cs *CommitSvc) Stop(ctx context.Context) error {
	log.Info("stop commit service")
	cs.host.RemoveStreamHandler(ShardStoreProtocol)
	return nil
}

func (cs *CommitSvc) handleShardStore(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req ShardStoreReq
	err := req.Unmarshal(s, "json")
	if err != nil {
		// TODO: respond error
	}
	key := datastore.NewKey(fmt.Sprintf("order-%d", req.OrderId))
	contentBytes, err := cs.db.Get(cs.ctx, key)
	if err != nil {
		// TODO: respond error
	}
	var resp = &ShardStoreResp{
		OrderId: req.OrderId,
		Cid:     req.Cid,
		Content: contentBytes,
	}
	err = resp.Marshal(s, "json")
	if err != nil {
		// TODO: respond error
	}
}

func (cs *CommitSvc) Commit(ctx context.Context, creator string, orderMeta types.OrderMeta, content any) (*CommitResult, error) {
	if !orderMeta.TxSent {
		orderId, txId, err := cs.chainSvc.Store(cs.nodeAddress, creator, cs.nodeAddress, orderMeta.Duration, orderMeta.Replica)
		if err != nil {
			return nil, err
		}
		log.Info("Store tx succeed. orderid=%d tx=%s", orderId, txId)
		orderMeta.OrderId = orderId
		orderMeta.TxId = txId
		orderMeta.TxSent = true
	}

	// TODO: store data.
	key := datastore.NewKey(fmt.Sprintf("order-%d", orderMeta.OrderId))
	cs.db.Put(ctx, key, []byte(content.(string)))

	// TODO: add timeout
	orderCompleteQuery := chain.QueryOrderComplete(orderMeta.OrderId)
	log.Debugf("subscribe query: %s", orderCompleteQuery)
	events, err := cs.chainListener.Subscribe(ctx, subscriber_gateway, orderCompleteQuery)
	if err != nil {
		return nil, err
	}
	e := <-events
	log.Info(e)
	// TODO: how to events chan close?

	log.Debugf("unsubscribe query: %s", orderCompleteQuery)
	err = cs.chainListener.Unsubscribe(ctx, subscriber_gateway, orderCompleteQuery)
	if err != nil {
		log.Error(err)
	}

	// TODO: get resource id from order

	return &CommitResult{
		OrderId: orderMeta.OrderId,
	}, nil
}
