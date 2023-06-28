package chain

import (
	"context"
	"sao-node/types"
	"time"

	sdkquerytypes "github.com/cosmos/cosmos-sdk/types/query"

	ordertypes "github.com/SaoNetwork/sao/x/order/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/ipfs/go-cid"
)

const (
	subscriber = "saonode"
	Blocktime  = 1 * time.Second
)

type OrderCompleteResult struct {
	Result string
}

func (c *ChainSvc) OrderReady(ctx context.Context, provider string, orderId uint64) (saotypes.MsgReadyResponse, string, int64, error) {
	txAddress := provider
	defer func() {
		if c.ap != nil && txAddress != provider {
			c.ap.SetAddressAvailable(txAddress)
		}
	}()

	var err error
	if c.ap != nil {
		txAddress, err = c.ap.GetRandomAddress(ctx)
		if err != nil {
			return saotypes.MsgReadyResponse{}, "", -1, types.Wrap(types.ErrAccountNotFound, err)
		}
	}

	_, err = c.cosmos.Account(txAddress)
	if err != nil {
		return saotypes.MsgReadyResponse{}, "", -1, types.Wrap(types.ErrAccountNotFound, err)
	}

	msg := &saotypes.MsgReady{
		OrderId:  orderId,
		Creator:  txAddress,
		Provider: provider,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(txAddress, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return saotypes.MsgReadyResponse{}, "", -1, types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return saotypes.MsgReadyResponse{}, "", -1, types.Wrapf(types.ErrTxProcessFailed, "MsgReady tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	var readyResp saotypes.MsgReadyResponse
	err = result.resp.Decode(&readyResp)
	if err != nil {
		return saotypes.MsgReadyResponse{}, "", -1, types.Wrapf(types.ErrTxProcessFailed, "failed to decode MsgReadyResponse, due to %v", err)
	}

	return readyResp, result.resp.TxResponse.TxHash, result.resp.TxResponse.Height, nil
}

func (c *ChainSvc) StoreOrder(ctx context.Context, signer string, clientProposal *types.OrderStoreProposal) (saotypes.MsgStoreResponse, string, int64, error) {
	txAddress := signer
	defer func() {
		if c.ap != nil && txAddress != signer {
			c.ap.SetAddressAvailable(txAddress)
		}
	}()

	var err error
	if c.ap != nil {
		txAddress, err = c.ap.GetRandomAddress(ctx)
		if err != nil {
			return saotypes.MsgStoreResponse{}, "", -1, types.Wrap(types.ErrAccountNotFound, err)
		}

		_, err = c.cosmos.Account(txAddress)
		if err != nil {
			return saotypes.MsgStoreResponse{}, "", -1, types.Wrap(types.ErrAccountNotFound, err)
		}
	}

	// TODO: Cid
	msg := &saotypes.MsgStore{
		Creator:  txAddress,
		Proposal: clientProposal.Proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: clientProposal.JwsSignature.Protected,
			Signature: clientProposal.JwsSignature.Signature,
		},
		Provider: signer,
	}

	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(txAddress, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return saotypes.MsgStoreResponse{}, "", -1, types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	// log.Debug("MsgStore result: ", txResp)
	if result.resp.TxResponse.Code != 0 {
		return saotypes.MsgStoreResponse{}, "", -1, types.Wrapf(types.ErrTxProcessFailed, "MsgStore tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	var storeResp saotypes.MsgStoreResponse
	err = result.resp.Decode(&storeResp)
	if err != nil {
		return saotypes.MsgStoreResponse{}, "", -1, types.Wrapf(types.ErrTxProcessFailed, "failed to decode MsgStoreResponse, due to %v", err)
	}
	return storeResp, result.resp.TxResponse.TxHash, result.resp.TxResponse.Height, nil
}

func (c *ChainSvc) CompleteOrder(ctx context.Context, creator string, orderId uint64, cid cid.Cid, size uint64) (string, int64, error) {
	txAddress := creator
	defer func() {
		if c.ap != nil && txAddress != creator {
			c.ap.SetAddressAvailable(txAddress)
		}
	}()

	var err error
	if c.ap != nil {
		txAddress, err = c.ap.GetRandomAddress(ctx)
		if err != nil {
			return "", -1, types.Wrap(types.ErrAccountNotFound, err)
		}
	}

	_, err = c.cosmos.Account(txAddress)
	if err != nil {
		return "", -1, types.Wrap(types.ErrAccountNotFound, err)
	}

	msg := &saotypes.MsgComplete{
		Creator:  txAddress,
		OrderId:  orderId,
		Cid:      cid.String(),
		Size_:    size,
		Provider: creator,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(txAddress, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", -1, types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", -1, types.Wrapf(types.ErrTxProcessFailed, "MsgComplete tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, result.resp.TxResponse.Height, nil
}

func (c *ChainSvc) RenewOrder(ctx context.Context, creator string, orderRenewProposal types.OrderRenewProposal) (string, map[string]string, error) {
	txAddress := creator
	defer func() {
		if c.ap != nil && txAddress != creator {
			c.ap.SetAddressAvailable(txAddress)
		}
	}()

	var err error
	if c.ap != nil {
		txAddress, err = c.ap.GetRandomAddress(ctx)
		if err != nil {
			return "", nil, types.Wrap(types.ErrAccountNotFound, err)
		}

		_, err = c.cosmos.Account(txAddress)
		if err != nil {
			return "", nil, types.Wrap(types.ErrAccountNotFound, err)
		}
	}

	msg := &saotypes.MsgRenew{
		Creator:      txAddress,
		Proposal:     orderRenewProposal.Proposal,
		JwsSignature: orderRenewProposal.JwsSignature,
		Provider:     creator,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(txAddress, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", nil, types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", nil, types.Wrapf(types.ErrTxProcessFailed, "MsgRenew tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	var renewResp saotypes.MsgRenewResponse
	err = result.resp.Decode(&renewResp)
	if err != nil {
		return "", nil, err
	}
	resultMap := make(map[string]string)
	for _, r := range renewResp.Result {
		resultMap[r.K] = r.V
	}
	return result.resp.TxResponse.TxHash, resultMap, nil
}

func (c *ChainSvc) MigrateOrder(ctx context.Context, creator string, dataIds []string) (string, map[string]string, int64, error) {
	txAddress := creator
	defer func() {
		if c.ap != nil && txAddress != creator {
			c.ap.SetAddressAvailable(txAddress)
		}
	}()

	var err error
	if c.ap != nil {
		txAddress, err = c.ap.GetRandomAddress(ctx)
		if err != nil {
			return "", nil, -1, types.Wrap(types.ErrAccountNotFound, err)
		}

		_, err = c.cosmos.Account(txAddress)
		if err != nil {
			return "", nil, -1, types.Wrap(types.ErrAccountNotFound, err)
		}
	}

	msg := &saotypes.MsgMigrate{
		Creator:  txAddress,
		Data:     dataIds,
		Provider: creator,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(txAddress, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", nil, -1, types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", nil, -1, types.Wrapf(types.ErrTxProcessFailed, "MsgMigrate tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	var migrateResp saotypes.MsgMigrateResponse
	err = result.resp.Decode(&migrateResp)
	if err != nil {
		return "", nil, -1, err
	}
	resultMap := make(map[string]string)
	for _, r := range migrateResp.Result {
		resultMap[r.K] = r.V
	}
	return result.resp.TxResponse.TxHash, resultMap, result.resp.TxResponse.Height, nil
}

func (c *ChainSvc) TerminateOrder(ctx context.Context, creator string, terminateProposal types.OrderTerminateProposal) (string, error) {
	txAddress := creator
	defer func() {
		if c.ap != nil && txAddress != creator {
			c.ap.SetAddressAvailable(txAddress)
		}
	}()

	var err error
	if c.ap != nil {
		txAddress, err = c.ap.GetRandomAddress(ctx)
		if err != nil {
			return "", types.Wrap(types.ErrAccountNotFound, err)
		}

		_, err = c.cosmos.Account(txAddress)
		if err != nil {
			return "", types.Wrap(types.ErrAccountNotFound, err)
		}
	}

	msg := &saotypes.MsgTerminate{
		Creator:      txAddress,
		Proposal:     terminateProposal.Proposal,
		JwsSignature: terminateProposal.JwsSignature,
		Provider:     creator,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(txAddress, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgTerminate tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) GetOrder(ctx context.Context, orderId uint64) (*ordertypes.FullOrder, error) {
	queryResp, err := c.orderClient.Order(ctx, &ordertypes.QueryGetOrderRequest{
		Id: orderId,
	})
	if err != nil {
		return nil, types.Wrap(types.ErrQueryOrderFailed, err)
	}
	return &queryResp.Order, nil
}

func (c *ChainSvc) GetShard(ctx context.Context, shardId uint64) (*ordertypes.Shard, error) {
	queryResp, err := c.orderClient.Shard(ctx, &ordertypes.QueryGetShardRequest{
		Id: shardId,
	})
	if err != nil {
		return nil, types.Wrap(types.ErrQueryShardFailed, err)
	}
	return &queryResp.Shard, nil
}

func (c *ChainSvc) ListShards(ctx context.Context, offset uint64, limit uint64) ([]ordertypes.Shard, uint64, error) {
	resp, err := c.orderClient.ShardAll(ctx, &ordertypes.QueryAllShardRequest{
		Pagination: &sdkquerytypes.PageRequest{Offset: offset, Limit: limit, Reverse: false, CountTotal: true}})

	if err != nil {
		return nil, 0, types.Wrap(types.ErrQueryOrderFailed, err)
	}
	return resp.Shard, resp.Pagination.Total, nil
}

// wsevent
//func (cs *ChainSvc) SubscribeOrderComplete(ctx context.Context, orderId uint64, doneChan chan OrderCompleteResult) error {
//	log.Debugf("SubscribeOrderComplete %s", QueryOrderComplete(orderId))
//	ch, err := cs.listener.Subscribe(ctx, subscriber, QueryOrderComplete(orderId))
//	if err != nil {
//		return err
//	}
//	log.Debugf("SubscribeOrderComplete %s succeed", QueryOrderComplete(orderId))
//
//	go func() {
//		log.Debugf("new thread wait chan")
//		<-ch
//		// TODO: replace with real data id.
//		// uuid, _ := uuid.GenerateUUID()
//		doneChan <- OrderCompleteResult{}
//		log.Debugf("new thread quit chan")
//	}()
//	return nil
//}
//
//func (cs *ChainSvc) UnsubscribeOrderComplete(ctx context.Context, orderId uint64) error {
//	err := cs.listener.Unsubscribe(ctx, subscriber, QueryOrderComplete(orderId))
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (cs *ChainSvc) SubscribeShardTask(ctx context.Context, nodeAddr string, shardTaskChan chan *ShardTask) error {
//	log.Debugf("SubscribeShardTask: %s", QueryOrderShard(nodeAddr))
//	ch, err := cs.listener.Subscribe(ctx, subscriber, QueryOrderShard(nodeAddr))
//	if err != nil {
//		return err
//	}
//
//	go func() {
//		for c := range ch {
//			log.Debugf("event: ", c)
//			providers := c.Events["new-shard.provider"]
//			var i int
//			for ii, provider := range providers {
//				if provider == nodeAddr {
//					i = ii
//					break
//				}
//			}
//			orderId, err := strconv.ParseUint(c.Events["new-shard.order-id"][i], 10, 64)
//			if err != nil {
//				log.Error(err)
//				continue
//			}
//			gateway := c.Events["new-shard.peer"][i]
//			shardCid := c.Events["new-shard.cid"][i]
//			operation := c.Events["new-shard.operation"][i]
//			cid, err := cid.Decode(shardCid)
//			if err != nil {
//				log.Error(err)
//				continue
//			}
//
//			order, err := cs.GetOrder(ctx, orderId)
//			if err != nil {
//				log.Error(err)
//				continue
//			}
//
//			shardTaskChan <- &ShardTask{
//				Owner:          order.Owner,
//				OrderId:        orderId,
//				Gateway:        gateway,
//				Cid:            cid,
//				OrderOperation: fmt.Sprintf("%d", order.Operation),
//				ShardOperation: operation,
//			}
//		}
//		log.Info("shard task loop ends.")
//	}()
//	return nil
//}
//
//func (cs *ChainSvc) UnsubscribeShardTask(ctx context.Context, nodeAddr string) error {
//	log.Debug("UnsubscribeShardTask")
//	err := cs.listener.Unsubscribe(ctx, subscriber, QueryOrderShard(nodeAddr))
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func QueryOrderShard(addr string) string {
//	return fmt.Sprintf("new-shard.provider='%s'", addr)
//}
//
//func QueryOrderComplete(orderId uint64) string {
//	return fmt.Sprintf("order-completed.order-id=%d", orderId)
//}
