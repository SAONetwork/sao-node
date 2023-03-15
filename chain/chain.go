package chain

import (
	"context"
	"encoding/hex"
	"sao-node/types"
	"time"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/SaoNetwork/sao-did/sid"
	didtypes "github.com/SaoNetwork/sao/x/did/types"
	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	ordertypes "github.com/SaoNetwork/sao/x/order/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/tendermint/tendermint/rpc/client/http"
)

var log = logging.Logger("chain")

const ADDRESS_PREFIX = "sao"

// chain service provides access to cosmos chain, mainly including tx broadcast, data query, event listen.
type ChainSvc struct {
	cosmos      cosmosclient.Client
	bankClient  banktypes.QueryClient
	orderClient ordertypes.QueryClient
	nodeClient  nodetypes.QueryClient
	didClient   didtypes.QueryClient
	modelClient modeltypes.QueryClient
	listener    *http.HTTP
}

type ChainSvcApi interface {
	Stop(ctx context.Context) error
	GetLastHeight(ctx context.Context) (int64, error)
	GetBalance(ctx context.Context, address string) (sdktypes.Coins, error)
	ShowDidInfo(ctx context.Context, did string)
	GetSidDocument(ctx context.Context, versionId string) (*sid.SidDocument, error)
	UpdateDidBinding(ctx context.Context, creator string, did string, accountId string) (string, error)
	QueryMetadata(ctx context.Context, req *types.MetadataProposal, height int64) (*saotypes.QueryMetadataResponse, error)
	GetMeta(ctx context.Context, dataId string) (*modeltypes.QueryGetMetadataResponse, error)
	UpdatePermission(ctx context.Context, signer string, proposal *types.PermissionProposal) (string, error)
	Create(ctx context.Context, creator string) (string, error)
	Reset(ctx context.Context, creator string, peerInfo string, status uint32) (string, error)
	GetNodePeer(ctx context.Context, creator string) (string, error)
	GetNodeStatus(ctx context.Context, creator string) (uint32, error)
	ListNodes(ctx context.Context) ([]nodetypes.Node, error)
	StartStatusReporter(ctx context.Context, creator string, status uint32)
	OrderReady(ctx context.Context, provider string, orderId uint64) (saotypes.MsgReadyResponse, string, int64, error)
	StoreOrder(ctx context.Context, signer string, clientProposal *types.OrderStoreProposal) (saotypes.MsgStoreResponse, string, int64, error)
	CompleteOrder(ctx context.Context, creator string, orderId uint64, cid cid.Cid, size uint64) (string, int64, error)
	RenewOrder(ctx context.Context, creator string, orderRenewProposal types.OrderRenewProposal) (string, map[string]string, error)
	MigrateOrder(ctx context.Context, creator string, dataIds []string) (string, map[string]string, int64, error)
	GetOrder(ctx context.Context, orderId uint64) (*ordertypes.FullOrder, error)
	//SubscribeOrderComplete(ctx context.Context, orderId uint64, doneChan chan OrderCompleteResult) error
	//UnsubscribeOrderComplete(ctx context.Context, orderId uint64) error
	//SubscribeShardTask(ctx context.Context, nodeAddr string, shardTaskChan chan *ShardTask) error
	//UnsubscribeShardTask(ctx context.Context, nodeAddr string) error
	TerminateOrder(ctx context.Context, creator string, terminateProposal types.OrderTerminateProposal) (string, error)
	GetTx(ctx context.Context, hash string, heigth int64) (*coretypes.ResultTx, error)
}

func NewChainSvc(
	ctx context.Context,
	chainAddress string,
	wsEndpoint string,
	keyringHome string,
) (*ChainSvc, error) {
	log.Debugf("initialize chain client")

	cosmos, err := cosmosclient.New(ctx,
		cosmosclient.WithAddressPrefix(ADDRESS_PREFIX),
		cosmosclient.WithNodeAddress(chainAddress),
		cosmosclient.WithKeyringDir(keyringHome),
		cosmosclient.WithGas("auto"),
	)
	if err != nil {
		return nil, types.Wrap(types.ErrCreateChainServiceFailed, err)
	}

	bankClient := banktypes.NewQueryClient(cosmos.Context())
	orderClient := ordertypes.NewQueryClient(cosmos.Context())
	nodeClient := nodetypes.NewQueryClient(cosmos.Context())
	didClient := didtypes.NewQueryClient(cosmos.Context())
	modelClient := modeltypes.NewQueryClient(cosmos.Context())

	log.Debugf("initialize chain listener")
	http, err := http.New(chainAddress, wsEndpoint)
	if err != nil {
		return nil, types.Wrap(types.ErrCreateChainServiceFailed, err)
	}
	// log.Debug("initialize chain listener2", chainAddress)

	// err = http.Reset()
	// if err != nil {
	// 	return nil, err
	// }
	// log.Debugf("initialize chain listener3")

	return &ChainSvc{
		cosmos:      cosmos,
		bankClient:  bankClient,
		orderClient: orderClient,
		nodeClient:  nodeClient,
		didClient:   didClient,
		modelClient: modelClient,
		listener:    http,
	}, nil
}

func (c *ChainSvc) Stop(ctx context.Context) error {
	if c.listener != nil {
		log.Infof("Stop chain listener.")
		err := c.listener.Stop()
		if err != nil {
			return types.Wrap(types.ErrStopChainServiceFailed, err)
		}
	}
	return nil
}

func (c *ChainSvc) GetLastHeight(ctx context.Context) (int64, error) {
	return c.cosmos.LatestBlockHeight(ctx)
}

func (c *ChainSvc) GetBalance(ctx context.Context, address string) (sdktypes.Coins, error) {
	return c.cosmos.BankBalances(ctx, address, nil)
}

func (c *ChainSvc) GetTx(ctx context.Context, hash string, height int64) (*coretypes.ResultTx, error) {
	for {
		curHeight, err := c.GetLastHeight(ctx)
		if err != nil {
			return nil, types.Wrap(types.ErrTxQueryFailed, err)
		}
		if curHeight > height {
			break
		}
		time.Sleep(time.Second)
	}
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return nil, types.Wrap(types.ErrTxQueryFailed, err)
	}
	return c.cosmos.RPC.Tx(ctx, hashBytes, true)
}
