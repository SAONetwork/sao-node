package chain

import (
	"context"

	didtypes "github.com/SaoNetwork/sao/x/did/types"
	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	ordertypes "github.com/SaoNetwork/sao/x/order/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	logging "github.com/ipfs/go-log/v2"
	"github.com/tendermint/tendermint/rpc/client/http"
)

var log = logging.Logger("chain")

// chain service provides access to cosmos chain, mainly including tx broadcast, data query, event listen.
type ChainSvc struct {
	cosmos      cosmosclient.Client
	orderClient ordertypes.QueryClient
	nodeClient  nodetypes.QueryClient
	modelClient modeltypes.QueryClient
	didClient   didtypes.QueryClient
	listener    *http.HTTP
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
	modelClient := modeltypes.NewQueryClient(cosmos.Context())
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
		modelClient: modelClient,
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
