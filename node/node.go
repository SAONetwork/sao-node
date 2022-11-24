package node

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sao-storage-node/api"
	"sao-storage-node/chain"
	"sao-storage-node/node/gateway"
	"sao-storage-node/node/transport"
	"sao-storage-node/store"
	"sort"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	saotypes "github.com/SaoNetwork/sao-did/types"
	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/mitchellh/go-homedir"
	did "github.com/ockam-network/did"

	"fmt"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/node/config"
	"sao-storage-node/node/model"
	"sao-storage-node/node/repo"
	"sao-storage-node/node/storage"
	"sao-storage-node/types"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"

	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"
)

var log = logging.Logger("node")

const NODE_STATUS_NA uint32 = 0
const NODE_STATUS_ONLINE uint32 = 1
const NODE_STATUS_SERVE_GATEWAY uint32 = 1 << 1
const NODE_STATUS_SERVE_STORAGE uint32 = 1 << 2
const NODE_STATUS_ACCEPT_ORDER uint32 = 1 << 3

type Node struct {
	ctx       context.Context
	cfg       *config.Node
	host      host.Host
	repo      *repo.Repo
	address   string
	stopFuncs []StopFunc
	// used by store module
	storeSvc  *storage.StoreSvc
	manager   *model.ModelManager
	tds       datastore.Read
	hfs       *gateway.HttpFileServer
	rpcServer *http.Server
}

type JwtPayload struct {
	Allow []auth.Permission
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

	listenAddrsOption := libp2p.ListenAddrStrings(cfg.Libp2p.ListenAddress...)
	host, err := libp2p.New(listenAddrsOption, libp2p.Identity(peerKey))
	if err != nil {
		return nil, err
	}

	peerInfos := ""
	for _, a := range host.Addrs() {
		withP2p := a.Encapsulate(multiaddr.StringCast("/p2p/" + host.ID().String()))
		log.Debug("addr=", withP2p.String())
		if len(peerInfos) > 0 {
			peerInfos = peerInfos + ","
		}
		peerInfos = peerInfos + withP2p.String()
	}

	// chain
	chainSvc, err := chain.NewChainSvc(ctx, cfg.Chain.AddressPrefix, cfg.Chain.Remote, cfg.Chain.WsEndpoint)
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

	sn := Node{
		ctx:       ctx,
		cfg:       cfg,
		repo:      repo,
		address:   nodeAddr,
		stopFuncs: stopFuncs,
		host:      host,
		tds:       tds,
	}

	for _, address := range cfg.Transport.TransportListenAddress {
		if strings.Contains(address, "udp") {
			_, err := transport.StartLibp2pRpcServer(ctx, &sn, address, peerKey, tds, cfg)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid transport server address %s", address)
		}
	}

	peerInfosBytes, err := tds.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	log.Info("Node Peer Information: ", string(peerInfosBytes))

	for _, ma := range strings.Split(string(peerInfosBytes), ",") {
		_, err := multiaddr.NewMultiaddr(ma)
		if err != nil {
			return nil, err
		}
	}

	var status = NODE_STATUS_ONLINE
	var storageManager *store.StoreManager = nil
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

		sn.storeSvc, err = storage.NewStoreService(ctx, nodeAddr, chainSvc, host, cfg.Transport.StagingPath, storageManager)

		log.Info("storage node initialized")
		if err != nil {
			return nil, err
		}
		go sn.storeSvc.Start(ctx)
		sn.stopFuncs = append(sn.stopFuncs, sn.storeSvc.Stop)
	}

	if cfg.Module.GatewayEnable {
		status = status | NODE_STATUS_SERVE_GATEWAY
		var gatewaySvc = gateway.NewGatewaySvc(ctx, nodeAddr, chainSvc, host, cfg, storageManager)
		sn.manager = model.NewModelManager(&cfg.Cache, gatewaySvc)
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

		log.Info("gateway node initialized")
	}

	// chainSvc.stop should be after chain listener unsubscribe
	sn.stopFuncs = append(sn.stopFuncs, chainSvc.Stop)

	_, err = chainSvc.Reset(ctx, sn.address, string(peerInfosBytes), status)
	if err != nil {
		return nil, err
	}

	chainSvc.StartStatusReporter(ctx, sn.address)

	return &sn, nil
}

func newRpcServer(ga api.GatewayApi, cfg *config.API) (*http.Server, error) {
	log.Info("initialize rpc server")

	handler, err := GatewayRpcHandler(ga, cfg.EnablePermission)
	if err != nil {
		return nil, xerrors.Errorf("failed to instantiate rpc handler: %w", err)
	}

	strma := strings.TrimSpace(cfg.ListenAddress)
	endpoint, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %s, %s", strma, err)
	}
	rpcServer, err := ServeRPC(handler, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to start json-rpc endpoint: %s", err)
	}
	return rpcServer, nil
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

func (n *Node) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	var payload JwtPayload
	key, err := n.repo.GetKeyBytes()
	if err != nil {
		return nil, err
	}

	if _, err := jwt.Verify([]byte(token), jwt.NewHS256(key), &payload); err != nil {
		return nil, xerrors.Errorf("JWT Verification failed: %w", err)
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
		return nil, err
	}
	return jwt.Sign(&p, jwt.NewHS256(key))
}

func (n *Node) Create(ctx context.Context, orderProposal types.OrderStoreProposal, orderId uint64, content []byte) (apitypes.CreateResp, error) {
	// verify signature
	didManager, err := saodid.NewDidManagerWithDid(orderProposal.Proposal.Owner)
	if err != nil {
		return apitypes.CreateResp{}, err
	}
	proposalBytes, err := orderProposal.Proposal.Marshal()
	if err != nil {
		return apitypes.CreateResp{}, err
	}

	_, err = didManager.VerifyJWS(saotypes.GeneralJWS{
		Payload: base64url.Encode(proposalBytes),
		Signatures: []saotypes.JwsSignature{
			saotypes.JwsSignature(orderProposal.JwsSignature),
		},
	})
	if err != nil {
		return apitypes.CreateResp{}, xerrors.Errorf("verify client order proposal signature failed: %v", err)
	}

	// model process
	model, err := n.manager.Create(ctx, orderProposal, orderId, content)
	if err != nil {
		return apitypes.CreateResp{}, err
	}

	return apitypes.CreateResp{
		Alias:  model.Alias,
		DataId: model.DataId,
		Cid:    model.Cid,
	}, nil
}

func (n *Node) CreateFile(ctx context.Context, orderProposal types.OrderStoreProposal, orderId uint64) (apitypes.CreateResp, error) {
	// Asynchronous order and the content has been uploaded already
	cidStr := orderProposal.Proposal.Cid
	key := datastore.NewKey(types.FILE_INFO_PREFIX + cidStr)
	if info, err := n.tds.Get(ctx, key); err == nil {
		var fileInfo *types.ReceivedFileInfo
		err := json.Unmarshal(info, &fileInfo)
		if err != nil {
			return apitypes.CreateResp{}, err
		}

		basePath, err := homedir.Expand(fileInfo.Path)
		if err != nil {
			return apitypes.CreateResp{}, err
		}

		var path = filepath.Join(basePath, cidStr)
		file, err := os.Open(path)
		if err != nil {
			return apitypes.CreateResp{}, err
		}

		content, err := io.ReadAll(file)
		if err != nil {
			return apitypes.CreateResp{}, err
		}

		model, err := n.manager.Create(ctx, orderProposal, orderId, content)
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
		return apitypes.CreateResp{}, xerrors.Errorf("invliad CID: %s", cidStr)
	}
}

func (n *Node) Load(ctx context.Context, req apitypes.LoadReq) (apitypes.LoadResp, error) {
	model, err := n.manager.Load(ctx, &req)
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

func (n *Node) Delete(ctx context.Context, owner string, key string, group string) (apitypes.DeleteResp, error) {
	model, err := n.manager.Delete(ctx, owner, key, group)
	if err != nil {
		return apitypes.DeleteResp{}, err
	}
	return apitypes.DeleteResp{
		DataId: model.DataId,
		Alias:  model.Alias,
	}, nil
}

func (n *Node) Update(ctx context.Context, orderProposal types.OrderStoreProposal, orderId uint64, patch []byte) (apitypes.UpdateResp, error) {
	// verify signature
	did, err := did.Parse(orderProposal.Proposal.Owner)
	if err != nil {
		return apitypes.UpdateResp{}, err
	}
	var resolver saotypes.DidResolver
	if did.Method == "key" {
		resolver = saokey.NewKeyResolver()
	}
	didManager := saodid.NewDidManager(nil, resolver)
	proposalBytes, err := orderProposal.Proposal.Marshal()
	if err != nil {
		return apitypes.UpdateResp{}, err
	}

	_, err = didManager.VerifyJWS(saotypes.GeneralJWS{
		Payload: base64url.Encode(proposalBytes),
		Signatures: []saotypes.JwsSignature{
			saotypes.JwsSignature(orderProposal.JwsSignature),
		},
	})
	if err != nil {
		return apitypes.UpdateResp{}, xerrors.Errorf("verify client order proposal signature failed: %v", err)
	}

	model, err := n.manager.Update(ctx, orderProposal, orderId, patch)
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

func (n *Node) Renew(ctx context.Context, timeout int32, renewModels map[string]uint64) error {
	return n.manager.Renew(ctx, timeout, renewModels)
}

func (n *Node) ShowCommits(ctx context.Context, owner string, key string, group string) (apitypes.ShowCommitsResp, error) {
	model, err := n.manager.ShowCommits(ctx, owner, key, group)
	if err != nil {
		return apitypes.ShowCommitsResp{}, err
	}
	return apitypes.ShowCommitsResp{
		DataId:  model.DataId,
		Alias:   model.Alias,
		Commits: model.Commits,
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
		return apitypes.GenerateTokenResp{}, xerrors.Errorf("failed to generate http file sever token")
	}
}

func (n *Node) GetHttpUrl(ctx context.Context, dataId string) (apitypes.GetUrlResp, error) {
	if n.cfg.SaoHttpFileServer.HttpFileServerAddress != "" {
		return apitypes.GetUrlResp{
			Url: "http://" + n.cfg.SaoHttpFileServer.HttpFileServerAddress + "/saonetwork/" + dataId,
		}, nil
	} else {
		return apitypes.GetUrlResp{}, xerrors.Errorf("failed to get http url")
	}
}

func (n *Node) GetIpfsUrl(ctx context.Context, cid string) (apitypes.GetUrlResp, error) {
	if n.cfg.SaoIpfs.Enable {
		return apitypes.GetUrlResp{
			Url: "ipfs+https://" + n.cfg.SaoHttpFileServer.HttpFileServerAddress + "/ipfs/" + cid,
		}, nil
	} else {
		return apitypes.GetUrlResp{}, xerrors.Errorf("failed to get ipfs url")
	}
}

func (n *Node) NodeAddress(ctx context.Context) (string, error) {
	return n.address, nil
}

func (n *Node) NetPeers(context.Context) ([]types.PeerInfo, error) {
	host := n.host
	conns := host.Network().Conns()
	out := make([]types.PeerInfo, len(conns))

	for i, conn := range conns {
		peer := conn.RemotePeer()
		info := types.PeerInfo{ID: peer}

		agent, err := host.Peerstore().Get(peer, "AgentVersion")
		if err == nil {
			info.Agent = agent.(string)
		}

		for _, a := range host.Peerstore().Addrs(peer) {
			info.Addrs = append(info.Addrs, a.String())
		}
		sort.Strings(info.Addrs)

		protocols, err := host.Peerstore().GetProtocols(peer)
		if err == nil {
			sort.Strings(protocols)
			info.Protocols = protocols
		}

		if cm := host.ConnManager().GetTagInfo(peer); cm != nil {
			info.ConnMgrMeta = &types.ConnMgrInfo{
				FirstSeen: cm.FirstSeen,
				Value:     cm.Value,
				Tags:      cm.Tags,
				Conns:     cm.Conns,
			}
		}

		out[i] = info
	}

	return out, nil
}
