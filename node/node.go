package node

import (
	"bufio"
	"context"
	"encoding/json"
	ip "github.com/SaoNetwork/sao-node/node/public_ip"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	saokey "github.com/SaoNetwork/sao-did/key"
	"github.com/SaoNetwork/sao-node/api"
	"github.com/SaoNetwork/sao-node/chain"
	"github.com/SaoNetwork/sao-node/node/gateway"
	"github.com/SaoNetwork/sao-node/node/indexer"
	"github.com/SaoNetwork/sao-node/node/indexer/gql"
	"github.com/SaoNetwork/sao-node/node/transport"
	"github.com/SaoNetwork/sao-node/store"
	"github.com/SaoNetwork/sao-node/utils"

	"cosmossdk.io/math"
	saodid "github.com/SaoNetwork/sao-did"
	"github.com/SaoNetwork/sao-did/sid"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"fmt"
	"strings"

	apitypes "github.com/SaoNetwork/sao-node/api/types"
	"github.com/SaoNetwork/sao-node/node/config"
	"github.com/SaoNetwork/sao-node/node/model"
	"github.com/SaoNetwork/sao-node/node/repo"
	"github.com/SaoNetwork/sao-node/node/storage"
	"github.com/SaoNetwork/sao-node/types"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("node")

const NODE_STATUS_NA uint32 = 0
const NODE_STATUS_ONLINE uint32 = 1
const NODE_STATUS_SERVE_GATEWAY uint32 = 1 << 1
const NODE_STATUS_SERVE_STORAGE uint32 = 1 << 2
const NODE_STATUS_ACCEPT_ORDER uint32 = 1 << 3
const NODE_STATUS_SERVE_INDEXER uint32 = 1 << 4

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
	indexSvc  *indexer.IndexSvc
}

type JwtPayload struct {
	Allow []auth.Permission
}

func NewNode(ctx context.Context, repo *repo.Repo, keyringHome string, cctx *cli.Context) (*Node, error) {
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
	if !cfg.Libp2p.ExternalIpEnable && !cfg.Libp2p.IntranetIpEnable && len(cfg.Libp2p.AnnounceAddresses) == 0 {
		cfg.Libp2p.ExternalIpEnable = true
		log.Warn("Intranet ip and external ip are both disabled, enable external ip as default")
	}

	// chain
	chainSvc, err := chain.NewChainSvc(ctx, cfg.Chain.Remote, cfg.Chain.WsEndpoint, keyringHome)
	if err != nil {
		return nil, err
	}

	if cfg.Libp2p.ExternalIpEnable && cfg.Libp2p.PublicAddress == "" {
		nodeList, err := chainSvc.ListNodes(ctx)
		if err != nil {
			return nil, err
		}
		cfg.Libp2p.PublicAddress = ip.DoPublicIpRequest(ctx, host, nodeList)
		if cfg.Libp2p.PublicAddress == "" {
			log.Warnf("failed to get external Ip")
		}
	}

	if len(cfg.Libp2p.AnnounceAddresses) > 0 {
		peerInfos = strings.Join(cfg.Libp2p.AnnounceAddresses, ",")
		for _, peerInfo := range strings.Split(peerInfos, ",") {
			_, err := multiaddr.NewMultiaddr(peerInfo)
			if err != nil {
				return nil, types.Wrapf(types.ErrInvalidPeerInfo, "%s", peerInfo)
			}
		}
	} else {
		for _, a := range host.Addrs() {
			withP2p := a.Encapsulate(multiaddr.StringCast("/p2p/" + host.ID().String()))
			if cfg.Libp2p.IntranetIpEnable {
				log.Debug("addr=", withP2p.String())
				if len(peerInfos) > 0 {
					peerInfos = peerInfos + ","
				}
				peerInfos = peerInfos + withP2p.String()
			}
			if cfg.Libp2p.ExternalIpEnable && cfg.Libp2p.PublicAddress != "" && strings.Contains(withP2p.String(), "127.0.0.1") {
				publicAddrWithP2p := strings.Replace(withP2p.String(), "127.0.0.1", cfg.Libp2p.PublicAddress, 1)
				log.Debug("addr=", publicAddrWithP2p)
				if len(peerInfos) > 0 {
					peerInfos = peerInfos + ","
				}
				peerInfos = peerInfos + publicAddrWithP2p
			}
		}
	}

	addresses := make([]string, 0)
	if cfg.Chain.TxPoolSize > 0 {
		ap, err := chain.LoadAddressPool(ctx, keyringHome, cfg.Chain.TxPoolSize)
		if err != nil {
			return nil, err
		}
		chainSvc.SetAddressPool(ctx, ap)

		for address := range ap.Addresses {
			addresses = append(addresses, address)
		}
	}

	var stopFuncs []StopFunc
	tds, err := repo.Datastore(ctx, "/transport")
	if err != nil {
		return nil, err
	}

	key := datastore.NewKey(fmt.Sprintf(types.PEER_INFO_PREFIX))
	tds.Put(ctx, key, []byte(peerInfos))
	if err != nil {
		return nil, err
	}
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

	transportStagingPath := path.Join(repo.Path, "staging")
	rpcHandler := transport.NewHandler(ctx, &sn, tds, cfg, transportStagingPath)
	for _, address := range cfg.Transport.TransportListenAddress {
		if strings.Contains(address, "udp") {
			_, err := transport.StartLibp2pRpcServer(ctx, address, peerKey, tds, cfg, rpcHandler)
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

	peerInfos = string(peerInfosBytes)
	if strings.HasSuffix(peerInfos, ",") {
		peerInfos = strings.TrimRight(peerInfos, ",")
		tds.Put(ctx, key, []byte(peerInfos))
		if err != nil {
			return nil, err
		}
	}

	log.Info("Node Peer Information: ", string(peerInfos))

	for _, ma := range strings.Split(peerInfos, ",") {
		_, err := multiaddr.NewMultiaddr(ma)
		if err != nil {
			return nil, types.Wrap(types.ErrInvalidServerAddress, err)
		}
	}

	var status = NODE_STATUS_ONLINE
	var storageManager *store.StoreManager = nil
	notifyChan := make(map[string]chan interface{})
	var backends []store.StoreBackend
	if cfg.SaoIpfs.Enable {
		ipfsPath := path.Join(repo.Path, "ipfs")
		ipfsDaemon, err := store.NewIpfsDaemon(ipfsPath)
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

	if cfg.Module.StorageEnable && cfg.Module.GatewayEnable {
		notifyChan[types.ShardAssignProtocol] = make(chan interface{})
		notifyChan[types.ShardCompleteProtocol] = make(chan interface{})
	}
	if cfg.Module.StorageEnable {
		status = status | NODE_STATUS_SERVE_STORAGE
		if cfg.Storage.AcceptOrder {
			status = status | NODE_STATUS_ACCEPT_ORDER
		}
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
				storageManager.AddBackend(ipfsBackend)
			}
		}

		sn.storeSvc, err = storage.NewStoreService(ctx, nodeAddr, chainSvc, host, transportStagingPath, storageManager, notifyChan, ods)
		if err != nil {
			return nil, err
		}
		log.Info("storage node initialized")
		go sn.storeSvc.Start(ctx)
		sn.stopFuncs = append(sn.stopFuncs, sn.storeSvc.Stop)
	}

	if cfg.Module.GatewayEnable {
		serverPath := cfg.SaoHttpFileServer.HttpFileServerPath
		if serverPath == "" {
			serverPath = path.Join(repo.Path, "http-files")
		}

		status = status | NODE_STATUS_SERVE_GATEWAY
		var gatewaySvc = gateway.NewGatewaySvc(ctx, nodeAddr, chainSvc, host, cfg, storageManager, notifyChan, ods, keyringHome, transportStagingPath, serverPath, rpcHandler)
		sn.manager = model.NewModelManager(&cfg.Cache, gatewaySvc)
		sn.gatewaySvc = gatewaySvc
		sn.stopFuncs = append(sn.stopFuncs, sn.manager.Stop)

		// http file server
		if cfg.SaoHttpFileServer.Enable {
			log.Info("initialize http file server")

			hfs, err := gateway.StartHttpFileServer(serverPath, &cfg.SaoHttpFileServer, cfg, cctx, keyringHome)
			if err != nil {
				return nil, err
			}
			sn.hfs = hfs
			sn.stopFuncs = append(sn.stopFuncs, hfs.Stop)
		}

		log.Info("gateway node initialized")
	}

	if cfg.Module.IndexerEnable {
		status = status | NODE_STATUS_SERVE_INDEXER
		jobsDs, err := repo.Datastore(ctx, "/indexer")
		if err != nil {
			return nil, err
		}

		dbPath, err := homedir.Expand(cfg.Indexer.DbPath)
		if err != nil {
			return nil, types.Wrap(types.ErrInvalidPath, err)
		}
		indexSvc := indexer.NewIndexSvc(ctx, chainSvc, jobsDs, dbPath)
		sn.indexSvc = indexSvc
		sn.stopFuncs = append(sn.stopFuncs, sn.indexSvc.Stop)

		graphqlServer := gql.NewGraphqlServer(cfg.Indexer.ListenAddress, indexSvc)
		err = graphqlServer.Start(ctx)
		if err != nil {
			return nil, err
		}
		sn.stopFuncs = append(sn.stopFuncs, graphqlServer.Stop)

		log.Info("indexing node initialized")
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

	reset := func(peerInfo string) (string, error) {
		if peerInfo != "" && len(peerInfosBytes) > 0 {
			peerInfo = string(peerInfosBytes) + "," + peerInfo
		} else {
			peerInfo = string(peerInfosBytes)
		}
		return chainSvc.Reset(ctx, sn.address, peerInfo, status, addresses, nil)
	}

	if cfg.Module.StorageEnable {
		// Connect to P2P network
		if cfg.Module.GatewayEnable {
			sn.ConnectToGatewayCluster(ctx, nil)
		} else {
			sn.ConnectToGatewayCluster(ctx, reset)
		}
	}

	// chainSvc.stop should be after chain listener unsubscribe
	sn.stopFuncs = append(sn.stopFuncs, chainSvc.Stop)

	hasPledged := false
	pledgeInfo, err := chainSvc.GetPledgeInfo(ctx, nodeAddr)
	if err != nil {
		if !strings.Contains(err.Error(), "code = NotFound desc = not found: key not found") {
			return nil, err
		}
	} else {
		fmt.Println(pledgeInfo)
		if pledgeInfo.Amount.GT(math.NewInt(0)) {
			hasPledged = true
		}
	}

	if !hasPledged {
		for {
			if !cfg.Module.StorageEnable || !cfg.Storage.AcceptOrder {
				break
			}

			fmt.Printf("Please make sure there is enough SAO tokens pledged for the storage in the account %s. Confirm with 'yes' :", nodeAddr)

			reader := bufio.NewReader(os.Stdin)
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return nil, types.Wrap(types.ErrInvalidParameters, err)
			}
			if strings.ToLower(strings.Replace(string(indata), "\n", "", -1)) != "yes" {
				continue
			}

			pledgeInfo, err := chainSvc.GetPledgeInfo(ctx, nodeAddr)
			if err != nil {
				if !strings.Contains(err.Error(), "code = NotFound desc = not found: key not found") {
					return nil, err
				} else {
					continue
				}
			} else {
				if pledgeInfo.Amount.GT(math.NewInt(0)) {
					break
				} else {
					continue
				}
			}
		}
	}

	if cfg.Module.GatewayEnable {
		reset("")
	}
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

func (n *Node) ConnectToGatewayCluster(ctx context.Context, reset func(string) (string, error)) {
	n.ConnectPeers(ctx, reset)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				n.ConnectPeers(ctx, reset)
				transport.DoPingRequest(ctx, n.host)

				log.Infof("Sent keep alive messages to peers")
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (n *Node) ConnectPeers(ctx context.Context, reset func(string) (string, error)) {
	nodes, err := n.chainSvc.ListNodes(ctx)
	if err != nil {
		log.Error(types.Wrap(types.ErrQueryNodeFailed, err))
		return
	}

	gateways := []*peer.AddrInfo{}
	for _, node := range nodes {
		if node.Status&NODE_STATUS_SERVE_GATEWAY == 0 {
			continue
		}

		if strings.Contains(node.Peer, n.host.ID().String()) {
			continue
		}

		for _, peerInfo := range strings.Split(node.Peer, ",") {
			if strings.Contains(peerInfo, "udp") || strings.Contains(peerInfo, "127.0.0.1") {
				continue
			}

			isFound := false
			for _, peer := range n.host.Network().Peers() {
				if strings.Contains(peerInfo, peer.ShortString()) {
					isFound = true
					break
				}
			}

			if isFound {
				continue
			}

			a, err := multiaddr.NewMultiaddr(peerInfo)
			if err != nil {
				log.Error(types.ErrInvalidServerAddress, "peerInfo=", peerInfo)
				continue
			}
			pi, err := peer.AddrInfoFromP2pAddr(a)
			if err != nil {
				log.Error(types.ErrInvalidServerAddress, "a=", a)
				continue
			}

			err = n.host.Connect(ctx, *pi)
			if err != nil {
				log.Info(types.ErrInvalidServerAddress, "a=", a)
				continue
			} else {
				log.Info("Connected to the peer ", node.Creator, " , peerinfos: ", node.Peer)
			}
			gateways = append(gateways, pi)
			break
		}
	}

	if reset != nil &&
		n.storeSvc != nil &&
		n.gatewaySvc == nil &&
		len(gateways) != 0 {

		ri, expiration := n.storeSvc.GetRelayInfo(ctx)
		if ri != "" {
			if time.Now().Before(expiration) {
				a, err := multiaddr.NewMultiaddr(ri)
				if err == nil {
					pi, err := peer.AddrInfoFromP2pAddr(a)
					if err == nil {
						if n.host.Network().Connectedness(pi.ID) == network.Connected {
							return
						}
					} else {
						log.Error(types.ErrInvalidPeerInfo, "multi address=", a, ", err: ", err)
					}
				} else {
					log.Error(types.ErrInvalidPeerInfo, "peerInfo=", ri, ", err: ", err)
				}
			}
		}

		var pi *peer.AddrInfo
		var reservation *client.Reservation
		for len(gateways) > 0 {
			pi = gateways[rand.Intn(len(gateways))]
			reservation, err = client.Reserve(context.Background(), n.host, *pi)
			if err == nil {
				break
			}
			log.Error(err)
		}
		if len(gateways) > 0 {
			ri = pi.Addrs[0].String() + "/p2p/" + pi.ID.String()
			n.storeSvc.SetRelayInfo(ctx, ri, reservation.Expiration)
			reset(ri + "/p2p-circuit/p2p/" + n.host.ID().String())
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
		Content:  model.Content,
	}, nil
}

func (n *Node) ModelLoadDelegate(ctx context.Context, req *types.MetadataProposal) (apitypes.LoadResp, error) {
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.LoadResp{}, err
	}

	keyringHome := os.Getenv("SAO_KEYRING_HOME")
	keyName := os.Getenv("SAO_KEY_NAME")
	didManager, _, err := n.getDidManager(ctx, keyringHome, keyName)
	if err != nil {
		return apitypes.LoadResp{}, err
	}

	req.Proposal.Owner = didManager.Id
	proposalBytes, err := req.Proposal.Marshal()
	if err != nil {
		return apitypes.LoadResp{}, types.Wrap(types.ErrMarshalFailed, err)
	}

	jws, err := didManager.CreateJWS(proposalBytes)
	if err != nil {
		return apitypes.LoadResp{}, types.Wrap(types.ErrCreateJwsFailed, err)
	}

	delegatedReq := &types.MetadataProposal{
		Proposal: req.Proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: jws.Signatures[0].Protected,
			Signature: jws.Signatures[0].Signature,
		},
	}

	model, err := n.manager.Load(ctx, delegatedReq)
	if err != nil {
		return apitypes.LoadResp{}, err
	}

	return apitypes.LoadResp{
		DataId:   model.DataId,
		Alias:    model.Alias,
		CommitId: model.CommitId,
		Version:  model.Version,
		Cid:      model.Cid,
		Content:  model.Content,
	}, nil
}

func (n *Node) ModelDelete(ctx context.Context, req *types.OrderTerminateProposal, isPublish bool) (apitypes.DeleteResp, error) {
	err := n.validSignature(ctx, &req.Proposal, req.Proposal.Owner, req.JwsSignature)
	if err != nil {
		return apitypes.DeleteResp{}, err
	}

	_, err = n.manager.Delete(ctx, req, isPublish)
	if err != nil {
		return apitypes.DeleteResp{}, err
	}
	return apitypes.DeleteResp{}, nil
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

	// log.Error("base64url.Encode(proposalBytes): ", base64url.Encode(proposalBytes))
	// log.Error("proposal: %#v", proposal)
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

func (n *Node) getDidManager(ctx context.Context, keyringHome string, keyName string) (*saodid.DidManager, string, error) {
	fmt.Println("keyName: ", keyName)

	address, err := chain.GetAddress(ctx, keyringHome, keyName)
	if err != nil {
		return nil, "", err
	}

	payload := fmt.Sprintf("cosmos %s allows to generate did", address)
	secret, err := chain.SignByAccount(ctx, keyringHome, keyName, []byte(payload))
	if err != nil {
		return nil, "", types.Wrap(types.ErrSignedFailed, err)
	}

	provider, err := saokey.NewSecp256k1Provider(secret)
	if err != nil {
		return nil, "", types.Wrap(types.ErrCreateProviderFailed, err)
	}
	resolver := saokey.NewKeyResolver()

	didManager := saodid.NewDidManager(provider, resolver)
	_, err = didManager.Authenticate([]string{}, "")
	if err != nil {
		return nil, "", types.Wrap(types.ErrAuthenticateFailed, err)
	}

	return &didManager, address, nil
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

func (n *Node) FaultsCheck(ctx context.Context, dataIds []string) (*apitypes.FileFaultsReportResp, error) {
	fishmen, err := n.chainSvc.GetFishmen(ctx)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(fishmen, n.address) {
		return nil, types.Wrapf(types.ErrInvalidParameters, "i am not a fishmen")
	}

	faultsMap := make(map[string][]*saotypes.Fault, 0)
	for _, dataId := range dataIds {
		meta, err := n.chainSvc.GetMeta(ctx, dataId)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		for provider, shard := range meta.Shards {

			passCheck := false

			result := make(chan types.ShardLoadResp)

			go func(result chan types.ShardLoadResp) {
				result <- n.gatewaySvc.FetchShard(ctx, provider, shard.Cid, shard.Peer, meta.Metadata.DataId, meta.Metadata.OrderId)
			}(result)

			select {
			case resp := <-result:
				if resp.Code == 0 {
					cid, err := utils.CalculateCid(resp.Content)
					if err != nil {
						log.Error(err.Error())
						continue
					}

					if cid.String() == meta.Metadata.Cid {
						passCheck = true
					}
				}

			case <-time.After(10 * time.Second):
				fmt.Println("Timeout")
			}

			if !passCheck {
				faults := faultsMap[provider]
				if faults == nil {
					faultsMap[provider] = make([]*saotypes.Fault, 0)
					faults = faultsMap[provider]
				}
				faultsMap[provider] = append(faults, &saotypes.Fault{
					DataId:   meta.Metadata.DataId,
					OrderId:  meta.Metadata.OrderId,
					ShardId:  shard.ShardId,
					CommitId: meta.Metadata.Commits[len(meta.Metadata.Commits)-1],
					Provider: provider,
					Reporter: n.address,
				})
			}
		}
	}

	for provider, faults := range faultsMap {
		if len(faults) > 0 {
			_, err := n.chainSvc.ReportFaults(ctx, n.address, provider, faults)
			if err != nil {
				log.Error(err.Error())
				delete(faultsMap, provider)
				continue
			}
		}
	}

	if len(faultsMap) > 0 {
		return &apitypes.FileFaultsReportResp{
			Faults: faultsMap,
		}, nil
	} else {
		return nil, types.Wrapf(types.ErrInvalidParameters, "no faults found")
	}
}

func (n *Node) RecoverCheck(ctx context.Context, provider string, faultIds []string) (*apitypes.FileRecoverReportResp, error) {
	fishmen, err := n.chainSvc.GetFishmen(ctx)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(fishmen, n.address) {
		return nil, types.Wrapf(types.ErrInvalidParameters, "i am not a fishmen")
	}

	recoverableFaults := make([]*saotypes.Fault, 0)
	for _, faultId := range faultIds {
		fault, err := n.chainSvc.GetFault(ctx, faultId)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		meta, err := n.chainSvc.GetMeta(ctx, fault.DataId)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		shardMeta, err := n.chainSvc.GetShard(ctx, fault.ShardId)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		peer := ""
		for provider, shard := range meta.Shards {
			if shard.ShardId == fault.ShardId && provider == fault.Provider {
				peer = shard.Peer
				break
			}
		}
		if peer == "" {
			log.Error("invalid shard ", fault.ShardId)
			continue
		}

		passCheck := false

		result := make(chan types.ShardLoadResp)

		go func(result chan types.ShardLoadResp) {
			result <- n.gatewaySvc.FetchShard(ctx, provider, shardMeta.Cid, peer, meta.Metadata.DataId, meta.Metadata.OrderId)
		}(result)

		select {
		case resp := <-result:
			if resp.Code == 0 {
				cid, err := utils.CalculateCid(resp.Content)
				if err != nil {
					log.Error(err.Error())
					continue
				}

				if cid.String() == meta.Metadata.Cid {
					passCheck = true
				}
			}

		case <-time.After(10 * time.Second):
			fmt.Println("Timeout")
		}

		if passCheck {
			commit := meta.Metadata.Commits[len(meta.Metadata.Commits)-1]
			commitId := strings.Split(commit, "\032")[0]
			recoverableFaults = append(recoverableFaults, &saotypes.Fault{
				DataId:   meta.Metadata.DataId,
				OrderId:  meta.Metadata.OrderId,
				ShardId:  fault.ShardId,
				CommitId: commitId,
				Provider: fault.Provider,
				Reporter: n.address,
			})
		}
	}

	if len(recoverableFaults) > 0 {
		_, err = n.chainSvc.RecoverFaults(ctx, n.address, provider, recoverableFaults)
		if err != nil {
			return nil, err
		}

		return &apitypes.FileRecoverReportResp{
			Faults: recoverableFaults,
		}, nil
	}

	return nil, types.Wrapf(types.ErrInvalidParameters, "no recoverable faults found")
}
