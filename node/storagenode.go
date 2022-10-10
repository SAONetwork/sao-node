package node

import (
	"context"
	"sao-storage-node/node/model"

	"fmt"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/node/config"
	"sao-storage-node/node/repo"
	"sao-storage-node/types"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/tendermint/tendermint/rpc/client/http"
	"golang.org/x/xerrors"
)

var log = logging.Logger("node")

type StorageNode struct {
	ctx             context.Context
	chainHttpClient *http.HTTP
	cfg             *config.StorageNode
	manager         *model.ModelManager
	host            *host.Host
	repo            *repo.Repo
	stopFuncs       []StopFunc
}

func NewStorageNode(ctx context.Context, repo *repo.Repo) (*StorageNode, error) {
	c, err := repo.Config()
	if err != nil {
		return nil, err
	}

	cfg, ok := c.(*config.StorageNode)
	if !ok {
		return nil, xerrors.Errorf("invalid config for repo, got: %T", c)
	}
	return &StorageNode{
		ctx:     ctx,
		cfg:     cfg,
		repo:    repo,
		manager: model.NewModelManager(&cfg.Cache),
	}, nil
}

func (n *StorageNode) Start() error {
	err := n.initChainWs()
	if err != nil {
		return err
	}

	err = n.initRpcServer()
	if err != nil {
		return err
	}

	//out, err := http.Subscribe(n.ctx, "", "node-login.creator='cosmos1angsar60505jnztcjxycwpmunsn5j7wl4f6rl3'")
	//if err != nil {
	//	return err
	//}
	//for o := range out {
	//	log.Infof("o: %v", o)
	//}
	return nil
}

func (n *StorageNode) initChainWs() error {
	log.Info("initialize tendermint websocket...")
	http, err := http.New(n.cfg.Chain.Remote, n.cfg.Chain.WsEndpoint)
	if err != nil {
		return err
	}
	err = http.Start()
	if err != nil {
		return err
	}
	n.chainHttpClient = http
	n.stopFuncs = append(n.stopFuncs, func(ctx context.Context) error {
		if n.chainHttpClient != nil {
			err = n.chainHttpClient.Stop()
			if err != nil {
				return err
			}
			log.Info("stop chain http client succeed.")
		}
		return nil
	})
	return nil
}

func (n *StorageNode) initRpcServer() error {
	log.Info("initialize rpc server")
	handler, err := GatewayRpcHandler(n)
	if err != nil {
		return xerrors.Errorf("failed to instantiate rpc handler: %w", err)
	}

	strma := strings.TrimSpace(n.cfg.Api.ListenAddress)
	endpoint, err := multiaddr.NewMultiaddr(strma)
	rpcStopper, err := ServeRPC(handler, endpoint)
	if err != nil {
		return fmt.Errorf("failed to start json-rpc endpoint: %s", err)
	}
	n.stopFuncs = append(n.stopFuncs, func(ctx context.Context) error {
		log.Info("stop rpc server succeed.")
		return rpcStopper(ctx)
	})
	return nil
}

func (n *StorageNode) Stop(ctx context.Context) error {
	for _, f := range n.stopFuncs {
		err := f(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *StorageNode) Test(ctx context.Context, msg string) (string, error) {
	return "world", nil
}

func (n *StorageNode) Create(ctx context.Context, orderMeta types.OrderMeta, commit any) (apitypes.CreateResp, error) {
	return apitypes.CreateResp{}, nil
}
