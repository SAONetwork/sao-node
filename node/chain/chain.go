package chain

import (
	"context"
	"fmt"
	"strconv"
	"time"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/hashicorp/go-uuid"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/tendermint/tendermint/rpc/client/http"
	"golang.org/x/xerrors"
)

var log = logging.Logger("chain")

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

// chain service provides access to cosmos chain, mainly including tx broadcast, data query, event listen.
type ChainSvc struct {
	cosmos   cosmosclient.Client
	listener *http.HTTP
}

func NewChainSvc(ctx context.Context, addressPrefix string, chainAddress string, wsEndpoint string) (*ChainSvc, error) {
	log.Infof("initialize chain client")
	cosmos, err := cosmosclient.New(ctx,
		cosmosclient.WithAddressPrefix(addressPrefix),
		cosmosclient.WithNodeAddress(chainAddress),
	)
	if err != nil {
		return nil, err
	}

	log.Info("initialize chain listener")
	http, err := http.New(chainAddress, wsEndpoint)
	if err != nil {
		return nil, err
	}
	err = http.Start()
	if err != nil {
		return nil, err
	}
	return &ChainSvc{
		cosmos:   cosmos,
		listener: http,
	}, nil
}

func (c *ChainSvc) Stop(ctx context.Context) error {
	if c.listener != nil {
		log.Infof("Stop chain listener.")
		err := c.listener.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainSvc) Login(ctx context.Context, creator string, ma multiaddr.Multiaddr, peerId peer.ID) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w", err)
	}

	msg := &nodetypes.MsgLogin{
		Creator: creator,
		Peer:    fmt.Sprintf("%v/p2p/%v", ma, peerId),
	}

	// TODO: recheck - seems BroadcastTx will return after confirmed on chain.
	txResp, err := c.cosmos.BroadcastTx(ctx, account, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgLogin tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Logout(ctx context.Context, creator string) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", err
	}

	msg := &nodetypes.MsgLogout{
		Creator: creator,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, account, msg)
	if err != nil {
		return "", err
	}

	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgLogout tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Reset(ctx context.Context, creator string, ma multiaddr.Multiaddr, peerId peer.ID) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", err
	}

	msg := &nodetypes.MsgReset{
		Creator: creator,
		Peer:    fmt.Sprintf("%v/p2p/%v", ma, peerId),
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, account, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgReset tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) OrderReady(ctx context.Context, provider string, orderId uint64) (string, error) {
	signerAcc, err := c.cosmos.Account(provider)
	if err != nil {
		return "", err
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

func (c *ChainSvc) StoreOrder(ctx context.Context, signer string, owner string, provider string, cid cid.Cid, duration int32, replica int32, metadata string) (uint64, string, error) {
	if signer != owner && signer != provider {
		return 0, "", xerrors.Errorf("Order tx signer must be owner or signer.")
	}

	signerAcc, err := c.cosmos.Account(signer)
	if err != nil {
		return 0, "", err
	}

	// TODO: Cid
	msg := &saotypes.MsgStore{
		Creator:  signer,
		Owner:    owner,
		Cid:      cid.String(),
		Provider: provider,
		Duration: duration,
		Replica:  replica,
		Metadata: metadata,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, signerAcc, msg)
	if err != nil {
		return 0, "", err
	}
	log.Debug("MsgStore result: ", txResp)
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
		return "", err
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

func (c *ChainSvc) GetOrder(ctx context.Context, orderId uint64) (*saotypes.Order, error) {
	queryClient := saotypes.NewQueryClient(c.cosmos.Context())
	queryResp, err := queryClient.Order(ctx, &saotypes.QueryGetOrderRequest{
		Id: orderId,
	})
	if err != nil {
		return nil, err
	}
	return &queryResp.Order, nil
}

func (c *ChainSvc) GetNodePeer(ctx context.Context, creator string) (string, error) {
	queryClient := nodetypes.NewQueryClient(c.cosmos.Context())
	resp, err := queryClient.Node(ctx, &nodetypes.QueryGetNodeRequest{
		Creator: creator,
	})
	if err != nil {
		return "", err
	}
	return resp.Node.Peer, nil
}

func (cs *ChainSvc) SubscribeOrderComplete(ctx context.Context, orderId uint64, doneChan chan OrderCompleteResult) error {
	ch, err := cs.listener.Subscribe(ctx, subscriber, QueryOrderComplete(orderId))
	if err != nil {
		return err
	}

	go func() {
		_ = <-ch
		// TODO: replace with real data id.
		uuid, _ := uuid.GenerateUUID()
		doneChan <- OrderCompleteResult{
			DataId: uuid,
		}
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

func (c *ChainSvc) QueryMeta(ctx context.Context, dataId string) (*modeltypes.QueryGetMetadataResponse, error) {
	queryClient := modeltypes.NewQueryClient(c.cosmos.Context())
	queryResp, err := queryClient.Metadata(ctx, &modeltypes.QueryGetMetadataRequest{
		DataId: dataId,
	})
	if err != nil {
		return nil, xerrors.Errorf("QueryMeta failed, " + err.Error())
	}
	return queryResp, nil
}

func (c *ChainSvc) QueryDataId(ctx context.Context, key string) (string, error) {
	queryClient := modeltypes.NewQueryClient(c.cosmos.Context())
	queryResp, err := queryClient.Model(ctx, &modeltypes.QueryGetModelRequest{
		Key: key,
	})
	if err != nil {
		return "", xerrors.Errorf("QueryDataId failed, " + err.Error())
	}
	return queryResp.Model.Data, nil
}

func QueryOrderShard(addr string) string {
	return fmt.Sprintf("new-shard.provider='%s'", addr)
}

func QueryOrderComplete(orderId uint64) string {
	return fmt.Sprintf("order-completed.order-id=%d", orderId)
}
