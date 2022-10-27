package node

import (
	"context"
	"sao-storage-node/api"
	"sao-storage-node/node/chain"
	"sao-storage-node/node/transport"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"

	"fmt"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/node/config"
	"sao-storage-node/node/model"
	"sao-storage-node/node/repo"
	"sao-storage-node/node/storage"
	"sao-storage-node/types"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"
)

var log = logging.Logger("node")

type Node struct {
	ctx context.Context
	cfg *config.Node
	//manager       *model.ModelManager
	host      *host.Host
	repo      *repo.Repo
	address   string
	stopFuncs []StopFunc
	// used by store module
	storeSvc *storage.StoreSvc
	manager  *model.ModelManager
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
	if err != nil {
		return nil, err
	}

	tds, err := repo.Datastore(ctx, "/transport")
	if err != nil {
		return nil, err
	}
	for _, address := range cfg.Transport.TransportListenAddress {
		if strings.Contains(address, "udp") {
			_, err := transport.StartTransportServer(ctx, address, peerKey, tds, cfg)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid transport server address %s", address)
		}
	}

	// chain
	chainSvc, err := chain.NewChainSvc(ctx, cfg.Chain.AddressPrefix, cfg.Chain.Remote, cfg.Chain.WsEndpoint)
	if err != nil {
		return nil, err
	}

	shardStaging := NewShardStaging(cfg.Transport.StagingPath)
	var stopFuncs []StopFunc

	sn := Node{
		ctx:       ctx,
		cfg:       cfg,
		repo:      repo,
		address:   nodeAddr,
		stopFuncs: stopFuncs,
		host:      &host,
	}

	if cfg.Module.GatewayEnable {
		// order db
		orderDb, err := repo.Datastore(ctx, "/order")
		if err != nil {
			return nil, err
		}

		ds, err := repo.Datastore(ctx, "/transport")
		if err != nil {
			return nil, err
		}

		sn.manager = model.NewModelManager(&cfg.Cache, storage.NewCommitSvc(ctx, nodeAddr, chainSvc, orderDb, host, &shardStaging), ds)
		sn.stopFuncs = append(sn.stopFuncs, sn.manager.Stop)

		// api server
		rpcStopper, err := newRpcServer(&sn, cfg.Api.ListenAddress)
		if err != nil {
			return nil, err
		}
		sn.stopFuncs = append(sn.stopFuncs, rpcStopper)
	}

	if cfg.Module.StorageEnable {
		sn.storeSvc, err = storage.NewStoreService(ctx, nodeAddr, chainSvc, host, &shardStaging)
		if err != nil {
			return nil, err
		}
		go sn.storeSvc.Start(ctx)
		sn.stopFuncs = append(sn.stopFuncs, sn.storeSvc.Stop)
	}

	// chainSvc.stop should be after chain listener unsubscribe
	sn.stopFuncs = append(sn.stopFuncs, chainSvc.Stop)

	return &sn, nil
}

func newRpcServer(ga api.GatewayApi, listenAddress string) (StopFunc, error) {
	log.Info("initialize rpc server")

	handler, err := GatewayRpcHandler(ga)
	if err != nil {
		return nil, xerrors.Errorf("failed to instantiate rpc handler: %w", err)
	}

	strma := strings.TrimSpace(listenAddress)
	endpoint, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %s, %s", strma, err)
	}
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
	model, err := n.manager.Create(ctx, orderMeta, types.ModelTypeFile)
	if err != nil {
		return apitypes.CreateResp{}, err
	}
	return apitypes.CreateResp{
		OrderId: model.OrderId,
		DataId:  model.DataId,
	}, nil
}

func (n *Node) CreateFile(ctx context.Context, orderMeta types.OrderMeta) (apitypes.CreateResp, error) {
	model, err := n.manager.Create(ctx, orderMeta, types.ModelTypeFile)
	if err != nil {
		return apitypes.CreateResp{}, err
	}
	return apitypes.CreateResp{
		OrderId: model.OrderId,
		DataId:  model.DataId,
	}, nil
}

func (n *Node) Load(ctx context.Context, owner string, alias string) (apitypes.LoadResp, error) {
	model, err := n.manager.Load(ctx, owner, alias)
	if err != nil {
		return apitypes.LoadResp{}, err
	}
	return apitypes.LoadResp{
		OrderId: model.OrderId,
		DataId:  model.DataId,
		Alias:   model.Alias,
		Content: model.Content,
	}, nil
}

func (n *Node) NodeAddress(ctx context.Context) (string, error) {
	return n.address, nil
}
