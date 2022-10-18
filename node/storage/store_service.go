package storage

import (
	"context"
	"sao-storage-node/node/chain"

	logging "github.com/ipfs/go-log/v2"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("storage")

type StoreSvc struct {
	nodeAddress string
	chainSvc    *chain.ChainSvc
	taskChan    chan *chain.ShardTask
	host        host.Host
}

func NewStoreService(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, host host.Host) (*StoreSvc, error) {
	ss := StoreSvc{
		nodeAddress: nodeAddress,
		chainSvc:    chainSvc,
		taskChan:    make(chan *chain.ShardTask),
		host:        host,
	}
	if err := ss.chainSvc.SubscribeShardTask(ctx, ss.nodeAddress, ss.taskChan); err != nil {
		return nil, err
	}

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
				log.Error(err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (ss *StoreSvc) process(ctx context.Context, task *chain.ShardTask) error {
	log.Infof("processing task: order id=%d gateway=%s shard_cid=%v", task.OrderId, task.Gateway, task.Cid)
	conn, err := ss.chainSvc.GetNodePeer(ctx, task.Gateway)
	if err != nil {
		return err
	}

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
	log.Infof("open stream(%s) to gateway %s", ShardStoreProtocol, conn)

	req := ShardStoreReq{
		OrderId: task.OrderId,
		Cid:     task.Cid,
	}
	log.Debugf("send ShardStoreReq: orderId=%d cid=%v", req.OrderId, req.Cid)
	var resp ShardStoreResp
	if err = DoRpc(ctx, stream, &req, &resp, "json"); err != nil {
		// TODO: handle error
		return err
	}
	// TODO: store resp.Content to ipfs
	log.Debugf("receive ShardStoreResp: content=%s", string(resp.Content))
	txHash, err := ss.chainSvc.CompleteOrder(ss.nodeAddress, task.OrderId, task.Cid, int32(len(resp.Content)))
	if err != nil {
		return err
	}
	log.Infof("Complete order succeed: %s", txHash)
	return nil
}

func (ss *StoreSvc) Stop(ctx context.Context) error {
	close(ss.taskChan)
	if err := ss.chainSvc.UnsubscribeShardTask(ctx, ss.nodeAddress); err != nil {
		return err
	}
	return nil
}
