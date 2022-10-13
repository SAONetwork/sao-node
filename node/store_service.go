package node

import (
	"context"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/tendermint/tendermint/rpc/client/http"
	"sao-storage-node/node/chain"
	"strconv"
)

const subscriber_storage = "storagenode"

type StoreSvc struct {
	nodeAddress   string
	chainListener *http.HTTP
	chainSvc      *chain.ChainSvc
	taskChan      chan *storeTask
	host          host.Host
}

type storeTask struct {
	orderId uint64
	gateway string
	cid     string
}

func NewStoreService(ctx context.Context, nodeAddress string, http *http.HTTP, chainSvc *chain.ChainSvc, host host.Host) *StoreSvc {
	ss := StoreSvc{
		nodeAddress:   nodeAddress,
		chainListener: http,
		chainSvc:      chainSvc,
		taskChan:      make(chan *storeTask),
		host:          host,
	}

	return &ss
}

func (ss *StoreSvc) StartProcessTasks(ctx context.Context) {
	for {
		select {
		case t := <-ss.taskChan:
			err := ss.process(ctx, t)
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (ss *StoreSvc) process(ctx context.Context, task *storeTask) error {
	log.Infof("processing task: order id=%d gateway=%s", task.orderId, task.gateway)
	conn, err := ss.chainSvc.GetNodePeer(ctx, task.gateway)
	if err != nil {
		return err
	}

	log.Infof("gateway peer id: %s", conn)
	a, err := ma.NewMultiaddr(conn)
	if err != nil {
		return err
	}
	pi, err := peer.AddrInfoFromP2pAddr(a)
	if err != nil {
		return err
	}
	err = ss.host.Connect(ctx, *pi)
	if err != nil {
		return err
	}
	stream, err := ss.host.NewStream(ctx, pi.ID, ShardStoreProtocol)
	if err != nil {
		return err
	}
	defer stream.Close()

	req := ShardStoreReq{
		OrderId: task.orderId,
		Cid:     task.cid,
	}
	var resp ShardStoreResp
	if err = DoRpc(ctx, stream, &req, &resp, "json"); err != nil {
		// TODO: handle error
		return err
	}
	// TODO: store resp.Content
	log.Debugf("resp.content: %s", string(resp.Content))
	txHash, err := ss.chainSvc.CompleteOrder(ctx, ss.nodeAddress, task.orderId, task.cid, int32(len(resp.Content)))
	if err != nil {
		return err
	}
	log.Infof("Complete order succeed: %s", txHash)
	return nil
}

func (ss *StoreSvc) SubscribeShardTask(ctx context.Context) error {
	queryOrderShard := chain.QueryOrderShard(ss.nodeAddress)
	log.Debugf("subscribe query: %v", queryOrderShard)
	reqchan, err := ss.chainListener.Subscribe(ctx, subscriber_storage, queryOrderShard)
	if err != nil {
		return err
	}
	go func() {
		for c := range reqchan {
			log.Infof("store chan: data: %v", c.Data)
			log.Infof("store chan: events: %v", c.Events)

			providers := c.Events["new-shard.provider"]
			var i int
			for ii, provider := range providers {
				if provider == ss.nodeAddress {
					i = ii
					break
				}
			}
			orderIdStr := c.Events["new-shard.order-id"][i]
			gateway := c.Events["new-shard.peer"][i]
			cid := c.Events["new-shard.cid"][i]

			orderId, err := strconv.ParseUint(orderIdStr, 10, 64)
			if err != nil {
				log.Error(err)
			} else {
				ss.taskChan <- &storeTask{
					orderId: orderId,
					gateway: gateway,
					cid:     cid,
				}
			}
		}
	}()
	return nil
}

func (ss *StoreSvc) UnsubscribeShardTask(ctx context.Context) error {
	queryOrderShard := chain.QueryOrderShard(ss.nodeAddress)
	log.Debugf("unsubscribe query: %s", queryOrderShard)
	if err := ss.chainListener.Unsubscribe(ctx, subscriber_storage, queryOrderShard); err != nil {
		return err
	}
	return nil
}
