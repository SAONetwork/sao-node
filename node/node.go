package node

import (
	"context"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"sao-storage-node/api"
	"sao-storage-node/node/chain"
	//"sao-storage-node/node/model"

	"fmt"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/node/config"
	"sao-storage-node/node/repo"
	"sao-storage-node/types"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"github.com/tendermint/tendermint/rpc/client/http"
	"golang.org/x/xerrors"
)

var log = logging.Logger("node")

type Node struct {
	ctx           context.Context
	chainListener *http.HTTP
	cfg           *config.Node
	//manager       *model.ModelManager
	host      *host.Host
	repo      *repo.Repo
	address   string
	commitSvc *CommitSvc
	storeSvc  *StoreSvc
	stopFuncs []StopFunc
}

func NewNode(ctx context.Context, repo *repo.Repo) (*Node, error) {
	c, err := repo.Config()
	if err != nil {
		return nil, err
	}

	cfg, ok := c.(*config.Node)
	if !ok {
		return nil, xerrors.Errorf("invalid config for repo, got: %T", c)
	}

	// get node address
	mds, err := repo.Datastore(ctx, "/metadata")
	if err != nil {
		return nil, err
	}
	abytes, err := mds.Get(ctx, datastore.NewKey("node-address"))
	if err != nil {
		return nil, err
	}
	nodeAddr := string(abytes)

	// p2p
	peerKey, err := repo.PeerId()
	if err != nil {
		return nil, err
	}
	listenAddrsOption := libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0")
	if len(cfg.Libp2p.ListenAddress) > 0 {
		listenAddrsOption = libp2p.ListenAddrStrings(cfg.Libp2p.ListenAddress...)
	}
	host, err := libp2p.New(listenAddrsOption, libp2p.Identity(peerKey))

	// chain
	chainSvc, err := chain.NewChainSvc(ctx, cfg.Chain.AddressPrefix, cfg.Chain.Remote)
	if err != nil {
		return nil, err
	}

	var stopFuncs []StopFunc
	// websocket subscriber_storage
	http, httpStopper, err := newChainWs(cfg.Chain.Remote, cfg.Chain.WsEndpoint)
	if err != nil {
		return nil, err
	}
	stopFuncs = append(stopFuncs, httpStopper)

	sn := Node{
		ctx:           ctx,
		cfg:           cfg,
		repo:          repo,
		address:       nodeAddr,
		chainListener: http,
		stopFuncs:     stopFuncs,
		host:          &host,
	}

	if cfg.Module.GatewayEnable {
		// order db
		orderDb, err := repo.Datastore(ctx, "/order")
		if err != nil {
			return nil, err
		}
		sn.commitSvc = NewCommitSvc(ctx, nodeAddr, chainSvc, http, orderDb, host)
		sn.commitSvc.Start()
		sn.stopFuncs = append(sn.stopFuncs, sn.commitSvc.Stop)

		rpcStopper, err := newRpcServer(&sn, cfg.Api.ListenAddress)
		if err != nil {
			return nil, err
		}
		sn.stopFuncs = append(sn.stopFuncs, rpcStopper)
	}
	if cfg.Module.StorageEnable {
		storeService := NewStoreService(ctx, nodeAddr, http, chainSvc, host)
		err = storeService.SubscribeShardTask(ctx)
		if err != nil {
			return nil, err
		}
		go storeService.StartProcessTasks(ctx)
		sn.storeSvc = storeService
		sn.stopFuncs = append(sn.stopFuncs, sn.storeSvc.UnsubscribeShardTask)
	}

	return &sn, nil
}

//func (n *Node) Start() error {
//	err := n.initChainWs()
//	if err != nil {
//		return err
//	}
//
//	err = n.initRpcServer()
//	if err != nil {
//		return err
//	}
//
//	//out, err := http.Subscribe(n.ctx, "", "node-login.creator='cosmos1angsar60505jnztcjxycwpmunsn5j7wl4f6rl3'")
//	//if err != nil {
//	//	return err
//	//}
//	//for o := range out {
//	//	log.Infof("o: %v", o)
//	//}
//	return nil
//}

func newChainWs(remote string, wsEndpoint string) (*http.HTTP, StopFunc, error) {
	log.Info("initialize tendermint websocket...")
	http, err := http.New(remote, wsEndpoint)
	if err != nil {
		return nil, nil, err
	}
	err = http.Start()
	if err != nil {
		return nil, nil, err
	}
	return http, func(ctx context.Context) error {
		if http != nil {
			err = http.Stop()
			if err != nil {
				return err
			}
			log.Info("stop chain http client succeed.")
		}
		return nil
	}, nil
}

func newRpcServer(ga api.GatewayApi, listenAddress string) (StopFunc, error) {
	log.Info("initialize rpc server")
	handler, err := GatewayRpcHandler(ga)
	if err != nil {
		return nil, xerrors.Errorf("failed to instantiate rpc handler: %w", err)
	}

	strma := strings.TrimSpace(listenAddress)
	endpoint, err := multiaddr.NewMultiaddr(strma)
	rpcStopper, err := ServeRPC(handler, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to start json-rpc endpoint: %s", err)
	}
	return func(ctx context.Context) error {
		log.Info("stop rpc server succeed.")
		return rpcStopper(ctx)
	}, nil
}

func (n *Node) Stop(ctx context.Context) error {
	for _, f := range n.stopFuncs {
		err := f(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) Test(ctx context.Context, msg string) (string, error) {
	return "world", nil
}

func (n *Node) Create(ctx context.Context, orderMeta types.OrderMeta, commit any) (apitypes.CreateResp, error) {
	n.commitSvc.Commit(ctx, orderMeta.Creator, orderMeta, commit)
	return apitypes.CreateResp{}, nil
}

func (n *Node) NodeAddress(ctx context.Context) (string, error) {
	return n.address, nil
}
