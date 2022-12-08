package chain

import (
	"context"
	"sao-storage-node/types"

	didtypes "github.com/SaoNetwork/sao/x/did/types"
	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	ordertypes "github.com/SaoNetwork/sao/x/order/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/tendermint/tendermint/rpc/client/http"
)

var log = logging.Logger("chain")

// chain service provides access to cosmos chain, mainly including tx broadcast, data query, event listen.
type ChainSvc struct {
	cosmos      cosmosclient.Client
	orderClient ordertypes.QueryClient
	nodeClient  nodetypes.QueryClient
	didClient   didtypes.QueryClient
	listener    *http.HTTP
}

type ChainSvcApi interface {
	Stop(ctx context.Context) error
	GetLastHeight(ctx context.Context) (int64, error)
	GetSidDocument(ctx context.Context, versionId string) (*didtypes.SidDocument, error)
	UpdateDidBinding(ctx context.Context, creator string, did string, accountId string) (string, error)
	Que3ryMeta(ctx context.Context, dataId string, height int64) (*modeltypes.QueryGetMetadataResponse, error)
	QueryDataId(ctx context.Context, key string) (string, error)
	QueryMetadata(ctx context.Context, req *types.MetadataProposal, height int64) (*saotypes.QueryMetadataResponse, error)
	UpdatePermission(ctx context.Context, signer string, proposal *types.PermissionProposal) (string, error)
	Login(ctx context.Context, creator string) (string, error)
	Logout(ctx context.Context, creator string) (string, error)
	Reset(ctx context.Context, creator string, peerInfo string, status uint32) (string, error)
	GetNodePeer(ctx context.Context, creator string) (string, error)
	GetNodeStatus(ctx context.Context, creator string) (uint32, error)
	StartStatusReporter(ctx context.Context, creator string, status uint32)
	OrderReady(ctx context.Context, provider string, orderId uint64) (string, error)
	StoreOrder(ctx context.Context, signer string, clientProposal *types.OrderStoreProposal) (uint64, string, error)
	CompleteOrder(ctx context.Context, creator string, orderId uint64, cid cid.Cid, size int32) (string, error)
	RenewOrder(ctx context.Context, creator string, orderRenewProposal types.OrderRenewProposal) (string, map[string]string, error)
	GetOrder(ctx context.Context, orderId uint64) (*ordertypes.Order, error)
	SubscribeOrderComplete(ctx context.Context, orderId uint64, doneChan chan OrderCompleteResult) error
	UnsubscribeOrderComplete(ctx context.Context, orderId uint64) error
	SubscribeShardTask(ctx context.Context, nodeAddr string, shardTaskChan chan *ShardTask) error
	UnsubscribeShardTask(ctx context.Context, nodeAddr string) error
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

	orderClient := ordertypes.NewQueryClient(cosmos.Context())
	nodeClient := nodetypes.NewQueryClient(cosmos.Context())
	didClient := didtypes.NewQueryClient(cosmos.Context())

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
		cosmos:      cosmos,
		orderClient: orderClient,
		nodeClient:  nodeClient,
		didClient:   didClient,
		listener:    http,
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

func (c *ChainSvc) GetLastHeight(ctx context.Context) (int64, error) {
	return c.cosmos.LatestBlockHeight(ctx)
}
