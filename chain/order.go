package chain

import (
	"context"
	"fmt"
	"sao-storage-node/types"
	"strconv"
	"time"

	ordertypes "github.com/SaoNetwork/sao/x/order/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

const (
	subscriber = "saonode"
	Blocktime  = 1 * time.Second
)

type ShardTask struct {
	Owner   string
	OrderId uint64
	Gateway string
	Cid     cid.Cid
}

type OrderCompleteResult struct {
	DataId string
}

func (c *ChainSvc) OrderReady(ctx context.Context, provider string, orderId uint64) (string, error) {
	signerAcc, err := c.cosmos.Account(provider)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w, check the keyring please", err)
	}

	msg := &saotypes.MsgReady{
		OrderId: orderId,
		Creator: provider,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, signerAcc, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgStore tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	dataResp := &saotypes.MsgReadyResponse{}
	err = txResp.Decode(dataResp)
	if err != nil {
		return "", err
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) StoreOrder(ctx context.Context, signer string, clientProposal types.ClientOrderProposal) (uint64, string, error) {
	//if signer != owner && signer != provider {
	//	return 0, "", xerrors.Errorf("Order tx signer must be owner or signer.")
	//}
	signerAcc, err := c.cosmos.Account(signer)
	if err != nil {
		return 0, "", xerrors.Errorf("%w, check the keyring please", err)
	}

	// TODO: Cid
	msg := &saotypes.MsgStore{
		Creator:  signer,
		Proposal: &clientProposal.Proposal,
		JwsSignature: &saotypes.JwsSignature{
			Protected: clientProposal.ClientSignature.Protected,
			Signature: clientProposal.ClientSignature.Signature,
		},
	}

	txResp, err := c.cosmos.BroadcastTx(ctx, signerAcc, msg)
	if err != nil {
		return 0, "", err
	}
	// log.Debug("MsgStore result: ", txResp)
	if txResp.TxResponse.Code != 0 {
		return 0, "", xerrors.Errorf("MsgStore tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	dataResp := &saotypes.MsgStoreResponse{}
	err = txResp.Decode(dataResp)
	if err != nil {
		return 0, "", err
	}
	return dataResp.OrderId, txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) CompleteOrder(ctx context.Context, creator string, orderId uint64, cid cid.Cid, size int32) (string, error) {
	signerAcc, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w, check the keyring please", err)
	}

	msg := &saotypes.MsgComplete{
		Creator: creator,
		OrderId: orderId,
		Cid:     cid.String(),
		Size_:   size,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, signerAcc, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgComplete tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) GetOrder(ctx context.Context, orderId uint64) (*ordertypes.Order, error) {
	queryResp, err := c.orderClient.Order(ctx, &ordertypes.QueryGetOrderRequest{
		Id: orderId,
	})
	if err != nil {
		return nil, err
	}
	return &queryResp.Order, nil
}

func (cs *ChainSvc) SubscribeOrderComplete(ctx context.Context, orderId uint64, doneChan chan OrderCompleteResult) error {
	ch, err := cs.listener.Subscribe(ctx, subscriber, QueryOrderComplete(orderId))
	if err != nil {
		return err
	}

	go func() {
		<-ch
		// TODO: replace with real data id.
		// uuid, _ := uuid.GenerateUUID()
		doneChan <- OrderCompleteResult{}
	}()
	return nil
}

func (cs *ChainSvc) UnsubscribeOrderComplete(ctx context.Context, orderId uint64) error {
	err := cs.listener.Unsubscribe(ctx, subscriber, QueryOrderComplete(orderId))
	if err != nil {
		return err
	}
	return nil
}

func (cs *ChainSvc) SubscribeShardTask(ctx context.Context, nodeAddr string, shardTaskChan chan *ShardTask) error {
	ch, err := cs.listener.Subscribe(ctx, subscriber, QueryOrderShard(nodeAddr))
	if err != nil {
		return err
	}

	go func() {
		for c := range ch {
			providers := c.Events["new-shard.provider"]
			var i int
			for ii, provider := range providers {
				if provider == nodeAddr {
					i = ii
					break
				}
			}
			orderId, err := strconv.ParseUint(c.Events["new-shard.order-id"][i], 10, 64)
			if err != nil {
				log.Error(err)
				continue
			}
			gateway := c.Events["new-shard.peer"][i]
			shardCid := c.Events["new-shard.cid"][i]
			cid, err := cid.Decode(shardCid)
			if err != nil {
				log.Error(err)
				continue
			}

			order, err := cs.GetOrder(ctx, orderId)
			if err != nil {
				log.Error(err)
				continue
			}

			shardTaskChan <- &ShardTask{
				Owner:   order.Owner,
				OrderId: orderId,
				Gateway: gateway,
				Cid:     cid,
			}
		}
	}()
	return nil
}

func (cs *ChainSvc) UnsubscribeShardTask(ctx context.Context, nodeAddr string) error {
	err := cs.listener.Unsubscribe(ctx, subscriber, QueryOrderShard(nodeAddr))
	if err != nil {
		return err
	}
	return nil
}

func QueryOrderShard(addr string) string {
	return fmt.Sprintf("new-shard.provider='%s'", addr)
}

func QueryOrderComplete(orderId uint64) string {
	return fmt.Sprintf("order-completed.order-id=%d", orderId)
}
