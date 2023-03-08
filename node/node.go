package node

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sao-node/api"
	"sao-node/chain"
	"sao-node/node/gateway"
	"sao-node/node/transport"
	"sao-node/store"
	"sort"

	saodid "github.com/SaoNetwork/sao-did"
	"github.com/SaoNetwork/sao-did/sid"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/mitchellh/go-homedir"

	"fmt"
	apitypes "sao-node/api/types"
	"sao-node/node/config"
	"sao-node/node/model"
	"sao-node/node/repo"
	"sao-node/node/storage"
	"sao-node/types"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/core/host"

	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("node")

const NODE_STATUS_NA uint32 = 0
const NODE_STATUS_ONLINE uint32 = 1
const NODE_STATUS_SERVE_GATEWAY uint32 = 1 << 1
const NODE_STATUS_SERVE_STORAGE uint32 = 1 << 2
const NODE_STATUS_ACCEPT_ORDER uint32 = 1 << 3

type Node struct {
	ctx        context.Context
	cfg        *config.Node
	host       host.Host
	repo       *repo.Repo
	address    string
	stopFuncs  []StopFunc
	gatewaySvc gateway.GatewaySvcApi
	// used by store module
	storeSvc  *storage.StoreSvc
	chainSvc  *chain.ChainSvc
	manager   *model.ModelManager
	tds       datastore.Read
	hfs       *gateway.HttpFileServer
	rpcServer *http.Server
}

type JwtPayload struct {
	Allow []auth.Permission
}

func NewNode(ctx context.Context, repo *repo.Repo, keyringHome string) (*Node, error) {
	c, err := repo.Config()
	if err != nil {
		return nil, err
	}

	cfg, ok := c.(*config.Node)
	if !ok {
		return nil, types.Wrapf(types.ErrDecodeConfigFailed, "invalid config for repo, got: %T", c)
	}

	// get node address
	mds, err := repo.Datastore(ctx, "/metadata")
	if err != nil {
		return nil, err
	}
	abytes, err := mds.Get(ctx, datastore.NewKey("node-address"))
	if err != nil {
		return nil, types.Wrap(types.ErrGetFailed, err)
	}
	nodeAddr := string(abytes)

	// p2p
	peerKey, err := repo.PeerId()
	if err != nil {
		return nil, err
	}

	listenAddrsOption := libp2p.ListenAddrStrings(cfg.Libp2p.ListenAddress...)
	host, err := libp2p.New(listenAddrsOption, libp2p.Identity(peerKey))
	if err != nil {
		return nil, types.Wrap(types.ErrCreateP2PServiceFaild, err)
	}

	peerInfos := ""
	if len(cfg.Libp2p.AnnounceAddresses) > 0 {
		peerInfos = strings.Join(cfg.Libp2p.AnnounceAddresses, ",")
	} else {
		for _, a := range host.Addrs() {
			withP2p := a.Encapsulate(multiaddr.StringCast("/p2p/" + host.ID().String()))
			log.Debug("addr=", withP2p.String())
			if len(peerInfos) > 0 {
				peerInfos = peerInfos + ","
			}
			peerInfos = peerInfos + withP2p.String()
		}
	}
	fmt.Println("cfg.Chain.Remote: ", cfg.Chain.Remote)
	// chain
	chainSvc, err := chain.NewChainSvc(ctx, cfg.Chain.Remote, cfg.Chain.WsEndpoint, keyringHome)
	if err != nil {
		return nil, err
	}

	var stopFuncs []StopFunc
	tds, err := repo.Datastore(ctx, "/transport")
	if err != nil {
		return nil, err
	}

	key := datastore.NewKey(fmt.Sprintf(types.PEER_INFO_PREFIX))
	tds.Put(ctx, key, []byte(peerInfos))

	ods, err := repo.Datastore(ctx, "/order")
	if err != nil {
		return nil, err
	}

	sn := Node{
		ctx:       ctx,
		cfg:       cfg,
		repo:      repo,
		address:   nodeAddr,
		stopFuncs: stopFuncs,
		host:      host,
		tds:       tds,
		chainSvc:  chainSvc,
	}

	for _, address := range cfg.Transport.TransportListenAddress {
		if strings.Contains(address, "udp") {
			_, err := transport.StartLibp2pRpcServer(ctx, &sn, address, peerKey, tds, cfg)
			if err != nil {
				return nil, types.Wrap(types.ErrStartLibP2PRPCServerFailed, err)
			}
		} else {
			return nil, types.Wrapf(types.ErrInvalidServerAddress, "invalid transport server address %s", address)
		}
	}

	peerInfosBytes, err := tds.Get(ctx, key)
	if err != nil {
		return nil, types.Wrap(types.ErrGetFailed, err)
	}
	log.Info("Node Peer Information: ", string(peerInfosBytes))

	for _, ma := range strings.Split(string(peerInfosBytes), ",") {
		_, err := multiaddr.NewMultiaddr(ma)
		if err != nil {
			return nil, types.Wrap(types.ErrInvalidServerAddress, err)
		}
	}

	var status = NODE_STATUS_ONLINE
	var storageManager *store.StoreManager = nil
	notifyChan := make(map[string]chan interface{})
	if cfg.Module.StorageEnable && cfg.Module.GatewayEnable {
		notifyChan[types.ShardAssignProtocol] = make(chan interface{})
		notifyChan[types.ShardCompleteProtocol] = make(chan interface{})
	}
	if cfg.Module.StorageEnable {
		status = status | NODE_STATUS_SERVE_STORAGE
		if cfg.Storage.AcceptOrder {
			status = status | NODE_STATUS_ACCEPT_ORDER
		}
		var backends []store.StoreBackend
		if len(cfg.Storage.Ipfs) > 0 {
			for _, f := range cfg.Storage.Ipfs {
				ipfsBackend, err := store.NewIpfsBackend(f.Conn, nil)
				if err != nil {
					return nil, err
				}
				err = ipfsBackend.Open()
				if err != nil {
					return nil, err
				}
				backends = append(backends, ipfsBackend)
			}
		}

		if cfg.SaoIpfs.Enable {
			ipfsDaemon, err := store.NewIpfsDaemon(cfg.SaoIpfs.Repo)
			if err != nil {
				return nil, err
			}
			daemonApi, node, err := ipfsDaemon.Start(ctx)
			if err != nil {
				return nil, err
			}
			sn.stopFuncs = append(sn.stopFuncs, func(_ context.Context) error {
				log.Info("close ipfs daemon.")
				return node.Close()
			})
			ipfsBackend, err := store.NewIpfsBackend("ipfs+sao", daemonApi)
			if err != nil {
				return nil, err
			}
			backends = append(backends, ipfsBackend)
			log.Info("ipfs daemon initialized")
		}

		storageManager = store.NewStoreManager(backends)
		log.Info("store manager daemon initialized")

		sn.storeSvc, err = storage.NewStoreService(ctx, nodeAddr, chainSvc, host, cfg.Transport.StagingPath, storageManager, notifyChan, ods)
		if err != nil {
			return nil, err
		}
		log.Info("storage node initialized")
		go sn.storeSvc.Start(ctx)
		sn.stopFuncs = append(sn.stopFuncs, sn.storeSvc.Stop)
	}

	if cfg.Module.GatewayEnable {
		status = status | NODE_STATUS_SERVE_GATEWAY
		var gatewaySvc = gateway.NewGatewaySvc(ctx, nodeAddr, chainSvc, host, cfg, storageManager, notifyChan, ods, keyringHome)
		sn.manager = model.NewModelManager(&cfg.Cache, gatewaySvc)
		sn.gatewaySvc = gatewaySvc
		sn.stopFuncs = append(sn.stopFuncs, sn.manager.Stop)

		// http file server
		if cfg.SaoHttpFileServer.Enable {
			log.Info("initialize http file server")

			hfs, err := gateway.StartHttpFileServer(&cfg.SaoHttpFileServer)
			if err != nil {
				return nil, err
			}
			sn.hfs = hfs
			sn.stopFuncs = append(sn.stopFuncs, hfs.Stop)
		}

		log.Info("gateway node initialized")
	}

	// api server
	rpcServer, err := newRpcServer(&sn, &cfg.Api)
	if err != nil {
		return nil, err
	}
	sn.rpcServer = rpcServer
	sn.stopFuncs = append(sn.stopFuncs, rpcServer.Shutdown)

	tokenRead, err := sn.AuthNew(ctx, api.AllPermissions[:2])
	if err != nil {
		return nil, err
	}
	log.Info("Read token: ", string(tokenRead))

	tokenWrite, err := sn.AuthNew(ctx, api.AllPermissions[:3])
	if err != nil {
		return nil, err
	}
	log.Info("Write token: ", string(tokenWrite))

	// Connect to P2P network
	sn.ConnectToGatewayCluster(ctx)

	// chainSvc.stop should be after chain listener unsubscribe
	sn.stopFuncs = append(sn.stopFuncs, chainSvc.Stop)

	_, err = chainSvc.Reset(ctx, sn.address, string(peerInfosBytes), status)
	log.Infof("repo: %s, Remote: %s, WsEndpoint： %s", repo.Path, cfg.Chain.Remote, cfg.Chain.WsEndpoint)
	log.Infof("node[%s] is joining SAO network...", sn.address)
	if err != nil {
		return nil, err
	}

	chainSvc.StartStatusReporter(ctx, sn.address, status)

	sn.stopFuncs = append(sn.stopFuncs, func(_ context.Context) error {
		for _, c := range notifyChan {
			close(c)
		}
		return nil
	})

	return &sn, nil
}

func newRpcServer(ga api.SaoApi, cfg *config.API) (*http.Server, error) {
	log.Info("initialize rpc server")

	handler, err := GatewayRpcHandler(ga, cfg.EnablePermission)
	if err != nil {
		return nil, types.Wrapf(types.ErrStartPRPCServerFailed, "failed to instantiate rpc handler: %v", err)
	}

	strma := strings.TrimSpace(cfg.ListenAddress)
	endpoint, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, types.Wrapf(types.ErrInvalidServerAddress, "invalid endpoint: %s, %s", strma, err)
	}
	rpcServer, err := ServeRPC(handler, endpoint)
	if err != nil {
		return nil, types.Wrapf(types.ErrStartPRPCServerFailed, "failed to start json-rpc endpoint: %s", err)
	}
	return rpcServer, nil
}

func (n *Node) ConnectToGatewayCluster(ctx context.Context) {
	nodes, err := n.chainSvc.ListNodes(ctx)
	if err != nil {
		log.Error(types.Wrap(types.ErrQueryNodeFailed, err))
		return
	}

	for _, node := range nodes {
		if node.Status&NODE_STATUS_SERVE_GATEWAY == 0 {
			continue
		}

		for _, peerInfo := range strings.Split(node.Peer, ",") {
			if strings.Contains(peerInfo, "udp") || strings.Contains(peerInfo, "127.0.0.1") {
				continue
			}

			a, err := multiaddr.NewMultiaddr(peerInfo)
			if err != nil {
				log.Error(types.ErrInvalidServerAddress, "peerInfo=%s", peerInfo)
				continue
			}
			pi, err := peer.AddrInfoFromP2pAddr(a)
			if err != nil {
				log.Error(types.ErrInvalidServerAddress, "a=%v", a)
				continue
			}

			err = n.host.Connect(ctx, *pi)
			if err != nil {
				log.Error(types.ErrInvalidServerAddress, "a=%v", a)
				continue
			} else {
				log.Info("Connected to the gateway ", node.Creator, " , peerinfos: ", node.Peer)
			}
			break
		}
	}
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

func (n *Node) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	var payload JwtPayload
	key, err := n.repo.GetKeyBytes()
	if err != nil {
		return nil, types.Wrap(types.ErrDecodeConfigFailed, err)
	}

	if _, err := jwt.Verify([]byte(token), jwt.NewHS256(key), &payload); err != nil {
		return nil, types.Wrapf(types.ErrInvalidJwt, "JWT Verification failed: %v", err)
	}

	log.Info("Permissions: ", payload)

	return payload.Allow, nil
}

func (n *Node) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	p := JwtPayload{
		Allow: perms, // TODO: consider checking validity
	}

	key, err := n.repo.GetKeyBytes()
	if err != nil {
		return nil, types.Wrap(types.ErrDecodeConfigFailed, err)
	}
	return jwt.Sign(&p, jwt.NewHS256(key))
}

func (n *Node) ModelCreate(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, content []byte) (apitypes.CreateResp, error) {
	// verify signature
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.CreateResp{}, err
	}

	err = n.validSignature(ctx, &orderProposal.Proposal, orderProposal.Proposal.Owner, orderProposal.JwsSignature)
	if err != nil {
		return apitypes.CreateResp{}, err
	}

	// model process
	model, err := n.manager.Create(ctx, req, orderProposal, orderId, content)
	if err != nil {
		return apitypes.CreateResp{}, err
	}

	return apitypes.CreateResp{
		Alias:  model.Alias,
		DataId: model.DataId,
		Cid:    model.Cid,
	}, nil
}

func (n *Node) ModelCreateFile(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64) (apitypes.CreateResp, error) {
	// Asynchronous order and the content has been uploaded already
	cidStr := orderProposal.Proposal.Cid
	key := datastore.NewKey(types.FILE_INFO_PREFIX + cidStr)
	if info, err := n.tds.Get(ctx, key); err == nil {
		var fileInfo *types.ReceivedFileInfo
		err := json.Unmarshal(info, &fileInfo)
		if err != nil {
			return apitypes.CreateResp{}, types.Wrap(types.ErrUnMarshalFailed, err)
		}

		basePath, err := homedir.Expand(fileInfo.Path)
		if err != nil {
			return apitypes.CreateResp{}, types.Wrap(types.ErrInvalidPath, err)
		}

		var path = filepath.Join(basePath, cidStr)
		file, err := os.Open(path)
		if err != nil {
			return apitypes.CreateResp{}, types.Wrap(types.ErrOpenFileFailed, err)
		}

		content, err := io.ReadAll(file)
		if err != nil {
			return apitypes.CreateResp{}, types.Wrap(types.ErrReadFileFailed, err)
		}

		// verify signature
		err = n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
		if err != nil {
			return apitypes.CreateResp{}, err
		}

		err = n.validSignature(ctx, &orderProposal.Proposal, orderProposal.Proposal.Owner, orderProposal.JwsSignature)
		if err != nil {
			return apitypes.CreateResp{}, err
		}

		model, err := n.manager.Create(ctx, req, orderProposal, orderId, content)
		if err != nil {
			return apitypes.CreateResp{}, err
		}
		return apitypes.CreateResp{
			Alias:  model.Alias,
			DataId: model.DataId,
			Cid:    model.Cid,
		}, nil
	} else {
		log.Error(err.Error())
		return apitypes.CreateResp{}, types.Wrapf(types.ErrInvalidCid, "invliad CID: %s", cidStr)
	}
}

func (n *Node) ModelLoad(ctx context.Context, req *types.MetadataProposal) (apitypes.LoadResp, error) {
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.LoadResp{}, err
	}

	model, err := n.manager.Load(ctx, req)
	if err != nil {
		return apitypes.LoadResp{}, err
	}

	return apitypes.LoadResp{
		DataId:   model.DataId,
		Alias:    model.Alias,
		CommitId: model.CommitId,
		Version:  model.Version,
		Cid:      model.Cid,
		Content:  string(model.Content),
	}, nil
}

func (n *Node) ModelDelete(ctx context.Context, req *types.OrderTerminateProposal, isPublish bool) (apitypes.DeleteResp, error) {
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.DeleteResp{}, err
	}

	model, err := n.manager.Delete(ctx, req, isPublish)
	if err != nil {
		return apitypes.DeleteResp{}, err
	}
	return apitypes.DeleteResp{
		DataId: model.DataId,
		Alias:  model.Alias,
	}, nil
}

func (n *Node) ModelUpdate(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, patch []byte) (apitypes.UpdateResp, error) {
	// verify signature
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.UpdateResp{}, err
	}

	err = n.validSignature(ctx, &orderProposal.Proposal, orderProposal.Proposal.Owner, orderProposal.JwsSignature)
	if err != nil {
		return apitypes.UpdateResp{}, err
	}

	model, err := n.manager.Update(ctx, req, orderProposal, orderId, patch)
	if err != nil {
		return apitypes.UpdateResp{}, err
	}
	return apitypes.UpdateResp{
		Alias:    model.Alias,
		DataId:   model.DataId,
		CommitId: model.CommitId,
		Cid:      model.Cid,
	}, nil
}

func (n *Node) ModelShowCommits(ctx context.Context, req *types.MetadataProposal) (apitypes.ShowCommitsResp, error) {
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.ShowCommitsResp{}, err
	}

	model, err := n.manager.ShowCommits(ctx, req)
	if err != nil {
		return apitypes.ShowCommitsResp{}, err
	}
	return apitypes.ShowCommitsResp{
		DataId:  model.DataId,
		Alias:   model.Alias,
		Commits: model.Commits,
	}, nil
}

func (n *Node) ModelRenewOrder(ctx context.Context, req *types.OrderRenewProposal, isPublish bool) (apitypes.RenewResp, error) {
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.RenewResp{}, err
	}

	results, err := n.manager.Renew(ctx, req, isPublish)
	if err != nil {
		return apitypes.RenewResp{}, err
	}
	return apitypes.RenewResp{
		Results: results,
	}, nil
}

func (n *Node) ModelUpdatePermission(ctx context.Context, req *types.PermissionProposal, isPublish bool) (apitypes.UpdatePermissionResp, error) {
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.UpdatePermissionResp{}, err
	}

	model, err := n.manager.UpdatePermission(ctx, req, isPublish)
	if err != nil {
		return apitypes.UpdatePermissionResp{}, err
	}
	return apitypes.UpdatePermissionResp{
		DataId: model.DataId,
	}, nil
}

func (n *Node) GetPeerInfo(ctx context.Context) (apitypes.GetPeerInfoResp, error) {
	key := datastore.NewKey(types.PEER_INFO_PREFIX)
	if peerInfo, err := n.tds.Get(ctx, key); err == nil {
		return apitypes.GetPeerInfoResp{
			PeerInfo: string(peerInfo),
		}, nil
	} else {
		return apitypes.GetPeerInfoResp{}, err
	}
}

func (n *Node) GenerateToken(ctx context.Context, owner string) (apitypes.GenerateTokenResp, error) {
	server, token := n.hfs.GenerateToken(owner)
	if token != "" {
		return apitypes.GenerateTokenResp{
			Server: server,
			Token:  token,
		}, nil
	} else {
		return apitypes.GenerateTokenResp{}, types.Wrapf(types.ErrGenerateTokenFaild, "failed to generate http file sever token")
	}
}

func (n *Node) GetHttpUrl(ctx context.Context, dataId string) (apitypes.GetUrlResp, error) {
	if n.cfg.SaoHttpFileServer.HttpFileServerAddress != "" {
		return apitypes.GetUrlResp{
			Url: "http://" + n.cfg.SaoHttpFileServer.HttpFileServerAddress + "/saonetwork/" + dataId,
		}, nil
	} else {
		return apitypes.GetUrlResp{}, types.Wrapf(types.ErrGetHttpUrlFaild, "failed to get http url")
	}
}

func (n *Node) GetIpfsUrl(ctx context.Context, cid string) (apitypes.GetUrlResp, error) {
	if n.cfg.SaoIpfs.Enable {
		return apitypes.GetUrlResp{
			Url: "ipfs+https://" + n.cfg.SaoHttpFileServer.HttpFileServerAddress + "/ipfs/" + cid,
		}, nil
	} else {
		return apitypes.GetUrlResp{}, types.Wrapf(types.ErrGetIpfsUrlFaild, "failed to get ipfs url")
	}
}

func (n *Node) GetNodeAddress(ctx context.Context) (string, error) {
	return n.address, nil
}

func (n *Node) GetNetPeers(context.Context) ([]types.PeerInfo, error) {
	host := n.host
	conns := host.Network().Conns()
	out := make([]types.PeerInfo, len(conns))

	for i, conn := range conns {
		peer := conn.RemotePeer()
		info := types.PeerInfo{ID: peer}

		for _, a := range host.Peerstore().Addrs(peer) {
			info.Addrs = append(info.Addrs, a.String())
		}
		sort.Strings(info.Addrs)

		out[i] = info
	}

	return out, nil
}

func (n *Node) getSidDocFunc() func(versionId string) (*sid.SidDocument, error) {
	return func(versionId string) (*sid.SidDocument, error) {
		return n.chainSvc.GetSidDocument(n.ctx, versionId)
	}
}

func (n *Node) validSignature(ctx context.Context, proposal types.ConsensusProposal, owner string, signature saotypes.JwsSignature) error {
	if owner == "all" {
		return nil
	}

	didManager, err := saodid.NewDidManagerWithDid(owner, n.getSidDocFunc())
	if err != nil {
		return types.Wrap(types.ErrInvalidDid, err)
	}

	proposalBytes, err := proposal.Marshal()
	if err != nil {
		return types.Wrap(types.ErrMarshalFailed, err)
	}

	log.Error("base64url.Encode(proposalBytes): ", base64url.Encode(proposalBytes))
	log.Error("proposal: %#v", proposal)
	_, err = didManager.VerifyJWS(saodidtypes.GeneralJWS{
		Payload: base64url.Encode(proposalBytes),
		Signatures: []saodidtypes.JwsSignature{
			saodidtypes.JwsSignature(signature),
		},
	})
	if err != nil {
		return types.Wrap(types.ErrInvalidSignature, err)
	}

	return nil
}

func (n *Node) OrderStatus(ctx context.Context, id string) (types.OrderInfo, error) {
	return n.gatewaySvc.OrderStatus(ctx, id)
}

func (n *Node) OrderList(ctx context.Context) ([]types.OrderInfo, error) {
	return n.gatewaySvc.OrderList(ctx)
}

func (n *Node) OrderFix(ctx context.Context, id string) error {
	return n.gatewaySvc.OrderFix(ctx, id)
}

func (n *Node) ShardStatus(ctx context.Context, orderId uint64, cid cid.Cid) (types.ShardInfo, error) {
	return n.storeSvc.ShardStatus(ctx, orderId, cid)
}

func (n *Node) ShardList(ctx context.Context) ([]types.ShardInfo, error) {
	return n.storeSvc.ShardList(ctx)
}

func (n *Node) ShardFix(ctx context.Context, orderId uint64, cid cid.Cid) error {
	return n.storeSvc.ShardFix(ctx, orderId, cid)
}

func (n *Node) ModelMigrate(ctx context.Context, dataIds []string) (apitypes.MigrateResp, error) {
	hash, results, err := n.storeSvc.Migrate(ctx, dataIds)
	return apitypes.MigrateResp{
		Results: results,
		TxHash:  hash,
	}, err
}

func (n *Node) MigrateJobList(ctx context.Context) ([]types.MigrateInfo, error) {
	return n.storeSvc.MigrateList(ctx)
}
