package chain

import (
	"context"
	"fmt"
	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/xerrors"
)

var log = logging.Logger("chain")

type ChainSvc struct {
	cosmos cosmosclient.Client
}

func NewChainSvc(ctx context.Context, addressPrefix string, chainAddress string) (*ChainSvc, error) {
	cosmos, err := cosmosclient.New(ctx,
		cosmosclient.WithAddressPrefix(addressPrefix),
		//cosmosclient.WithNodeAddress(chainAddress),
	)
	if err != nil {
		return nil, err
	}
	return &ChainSvc{
		cosmos: cosmos,
	}, nil
}

func (c *ChainSvc) Login(creator string, multiaddress string, peerId peer.ID) (string, error) {
	msg := &nodetypes.MsgLogin{
		Creator: creator,
		Peer:    fmt.Sprintf("%v/p2p/%v", multiaddress, peerId),
	}

	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w", err)
	}
	log.Infof(account.Name)
	log.Info(account.Address("cosmos"))

	// TODO: recheck - seems BroadcastTx will return after confirmed on chain.
	txResp, err := c.cosmos.BroadcastTx(account, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgLogin transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Logout(creator string) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", err
	}

	msg := &nodetypes.MsgLogout{
		Creator: creator,
	}
	txResp, err := c.cosmos.BroadcastTx(account, msg)
	if err != nil {
		return "", err
	}

	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgLogout transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	} else {
		log.Infof("MsgLogout transaction %v succeed.", txResp.TxResponse.TxHash)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Reset(creator string, multiaddr string, peerId peer.ID) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", err
	}

	// TODO: validate peer
	msg := &nodetypes.MsgReset{
		Creator: creator,
		Peer:    fmt.Sprintf("%v/p2p/%v", multiaddr, peerId),
	}
	txResp, err := c.cosmos.BroadcastTx(account, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgReset transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
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

func (c *ChainSvc) Store(signer string, creator string, provider string, duration int32, replica int32) (uint64, string, error) {
	if signer != creator && signer != provider {
		return 0, "", xerrors.Errorf("Order tx signer must be creator or signer.")
	}

	signerAcc, err := c.cosmos.Account(signer)
	if err != nil {
		return 0, "", err
	}

	msg := &saotypes.MsgStore{
		Creator:  creator,
		Cid:      "QmeSoArjthZ5VcaeJxg35rRPt6gwd4sWyPmNbYSpKtF4uF",
		Provider: provider,
		Duration: duration,
		Replica:  replica,
	}
	txResp, err := c.cosmos.BroadcastTx(signerAcc, msg)
	if err != nil {
		return 0, "", err
	}
	log.Debug("MsgStore result: ", txResp)
	if txResp.TxResponse.Code != 0 {
		return 0, "", xerrors.Errorf("MsgStore tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	} else {
		dataResp := &saotypes.MsgStoreResponse{}
		err = txResp.Decode(dataResp)
		if err != nil {
			return 0, "", err
		}
		log.Infof("MsgStore transaction %v succeed: orderId=%d", txResp.TxResponse.TxHash, dataResp.OrderId)
		return dataResp.OrderId, txResp.TxResponse.TxHash, nil
	}
}

func (c *ChainSvc) CompleteOrder(ctx context.Context, creator string, orderId uint64, cid string, size int32) (string, error) {
	signerAcc, err := c.cosmos.Account(creator)
	if err != nil {
		return "", err
	}

	msg := &saotypes.MsgComplete{
		Creator: creator,
		OrderId: orderId,
		Cid:     cid,
		Size_:   size,
	}
	txResp, err := c.cosmos.BroadcastTx(signerAcc, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgComplete tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	} else {
		dataResp := &saotypes.MsgCompleteResponse{}
		err = txResp.Decode(dataResp)
		if err != nil {
			return "", err
		}
		log.Infof("MsgComplete transaction %v succeed: orderId=%d", txResp.TxResponse.TxHash)
		return txResp.TxResponse.TxHash, nil
	}
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

func QueryOrderShard(addr string) string {
	return fmt.Sprintf("new-shard.provider='%s'", addr)
}

func QueryOrderComplete(orderId uint64) string {
	return fmt.Sprintf("order-completed.order-id=%d", orderId)
}
