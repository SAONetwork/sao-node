package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sao-node/chain"
	"sao-node/node/transport"
	"sao-node/store"
	"sao-node/types"
	"strings"
	"time"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"

	sid "github.com/SaoNetwork/sao-did/sid"
	logging "github.com/ipfs/go-log/v2"

	saodid "github.com/SaoNetwork/sao-did"
	saodidtypes "github.com/SaoNetwork/sao-did/types"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("storage")

type StoreSvc struct {
	nodeAddress  string
	chainSvc     *chain.ChainSvc
	taskChan     chan *chain.ShardTask
	host         host.Host
	stagingPath  string
	storeManager *store.StoreManager
	ctx          context.Context
}

func NewStoreService(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, host host.Host, stagingPath string, storeManager *store.StoreManager) (*StoreSvc, error) {
	ss := StoreSvc{
		nodeAddress:  nodeAddress,
		chainSvc:     chainSvc,
		taskChan:     make(chan *chain.ShardTask),
		host:         host,
		stagingPath:  stagingPath,
		storeManager: storeManager,
		ctx:          ctx,
	}

	host.SetStreamHandler(types.ShardLoadProtocol, ss.HandleShardStream)

	if err := ss.chainSvc.SubscribeShardTask(ctx, ss.nodeAddress, ss.taskChan); err != nil {
		return nil, err
	}

	return &ss, nil
}

func (ss *StoreSvc) Start(ctx context.Context) error {
	for {
		select {
		case t, ok := <-ss.taskChan:
			if !ok {
				return nil
			}
			err := ss.process(ctx, t)
			if err != nil {
				// TODO: retry mechanism
				log.Error(err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (ss *StoreSvc) process(ctx context.Context, task *chain.ShardTask) error {
	log.Debugf("processing task: order id=%d gateway=%s shard_cid=%v", task.OrderId, task.Gateway, task.Cid)
	log.Debugf("processing task: order operation=%s shard operation=%s", task.OrderOperation, task.ShardOperation)

	var shard []byte
	var err error

	// check if it's a renew order(Operation is 2)
	if task.OrderOperation != "2" || task.ShardOperation != "2" {
		// check if gateway is node itself
		if task.Gateway == ss.nodeAddress {
			shard, err = ss.getShardFromLocal(task.Owner, task.Cid)
			if err != nil {
				log.Warn("skip the known error: ", err.Error())
				return err
			}
		} else {
			shard, err = ss.getShardFromGateway(ctx, task.Owner, task.Gateway, task.OrderId, task.Cid)
			if err != nil {
				return err
			}
		}

		// store to backends
		_, err = ss.storeManager.Store(ctx, task.Cid, bytes.NewReader(shard))
		if err != nil {
			return err
		}
	} else {
		// make sure the data is still there
		isExist := ss.storeManager.IsExist(ctx, task.Cid)
		if !isExist {
			return xerrors.Errorf("shard with cid %s not found", task.Cid)
		}
	}

	log.Info("Complete order")
	txHash, err := ss.chainSvc.CompleteOrder(ctx, ss.nodeAddress, task.OrderId, task.Cid, int32(len(shard)))
	if err != nil {
		return err
	}
	log.Infof("Complete order succeed: txHash:%s, OrderId: %d, cid: %s", txHash, task.OrderId, task.Cid)
	return nil
}

func (ss *StoreSvc) getShardFromLocal(creator string, cid cid.Cid) ([]byte, error) {
	path, err := homedir.Expand(ss.stagingPath)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%v", cid)
	bytes, err := os.ReadFile(filepath.Join(path, creator, filename))
	if err != nil {
		return nil, err
	} else {
		return bytes, nil
	}
}

func (ss *StoreSvc) getShardFromGateway(ctx context.Context, owner string, gateway string, orderId uint64, cid cid.Cid) ([]byte, error) {
	peerInfos, err := ss.chainSvc.GetNodePeer(ctx, gateway)
	if err != nil {
		return nil, err
	}

	for _, peerInfo := range strings.Split(peerInfos, ",") {
		if strings.Contains(peerInfo, "udp") {
			continue
		}

		a, err := ma.NewMultiaddr(peerInfo)
		if err != nil {
			return nil, err
		}
		log.Info("conn: ", peerInfo)
		log.Info("a: ", peerInfo)
		pi, err := peer.AddrInfoFromP2pAddr(a)
		if err != nil {
			return nil, err
		}
		err = ss.host.Connect(ctx, *pi)
		if err != nil {
			return nil, err
		}
		stream, err := ss.host.NewStream(ctx, pi.ID, types.ShardStoreProtocol)
		if err != nil {
			return nil, err
		}
		defer stream.Close()
		log.Infof("open stream(%s) to gateway %s", types.ShardStoreProtocol, peerInfo)

		// Set a deadline on reading from the stream so it doesn't hang
		_ = stream.SetReadDeadline(time.Now().Add(300 * time.Second))
		defer stream.SetReadDeadline(time.Time{}) // nolint

		req := types.ShardReq{
			Owner:   owner,
			OrderId: orderId,
			Cid:     cid,
		}
		log.Debugf("send ShardReq: orderId=%d cid=%v", req.OrderId, req.Cid)
		var resp types.ShardResp
		if err = transport.DoRequest(ctx, stream, &req, &resp, "json"); err != nil {
			// TODO: handle error
			return nil, err
		}
		log.Debugf("receive ShardResp: content=%s", string(resp.Content))
		return resp.Content, nil
	}

	return nil, xerrors.Errorf("no valid node peer found")
}

func (ss *StoreSvc) Stop(ctx context.Context) error {
	close(ss.taskChan)
	if err := ss.chainSvc.UnsubscribeShardTask(ctx, ss.nodeAddress); err != nil {
		return err
	}
	return nil
}

func (ss *StoreSvc) getSidDocFunc() func(versionId string) (*sid.SidDocument, error) {
	return func(versionId string) (*sid.SidDocument, error) {
		return ss.chainSvc.GetSidDocument(ss.ctx, versionId)
	}
}

func (ss *StoreSvc) HandleShardStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardReq
	err := req.Unmarshal(s, "json")
	if err != nil {
		log.Error(err)
		// TODO: respond error
		return
	}
	log.Debugf("receive ShardReq: orderId=%d cid=%v", req.OrderId, req.Cid)

	didManager, err := saodid.NewDidManagerWithDid(req.Proposal.Proposal.Owner, ss.getSidDocFunc())
	if err != nil {
		log.Error(err)
		return
	}

	proposalBytes, err := req.Proposal.Proposal.Marshal()
	if err != nil {
		log.Error(err)
		return
	}

	_, err = didManager.VerifyJWS(saodidtypes.GeneralJWS{
		Payload: base64url.Encode(proposalBytes),
		Signatures: []saodidtypes.JwsSignature{
			saodidtypes.JwsSignature(req.Proposal.JwsSignature),
		},
	})
	if err != nil {
		log.Error("verify client order proposal signature failed: %v", err)
		return
	}

	lastHeight, err := ss.chainSvc.GetLastHeight(ss.ctx)
	if err != nil {
		log.Error(err)
		return
	}
	peerInfo := string(s.Conn().RemotePeer())
	if strings.Contains(req.Proposal.Proposal.Gateway, peerInfo) {
		log.Errorf("invalid query, unexpect gateway:%s, should be %s", peerInfo, req.Proposal.Proposal.Gateway)
		return
	}
	if req.Proposal.Proposal.LastValidHeight < uint64(lastHeight) {
		log.Errorf("invalid query, LastValidHeight:%d > now:%d", req.Proposal.Proposal.LastValidHeight, lastHeight)
		return
	}

	reader, err := ss.storeManager.Get(ss.ctx, req.Cid)
	if err != nil {
		log.Error(err)
		return
	}
	shardContent, err := io.ReadAll(reader)
	if err != nil {
		log.Error(err)
		return
	}

	var resp = &types.ShardResp{
		OrderId: req.OrderId,
		Cid:     req.Cid,
		Content: shardContent,
	}
	log.Debugf("send ShardResp: Content len %d", len(shardContent))

	err = resp.Marshal(s, "json")
	if err != nil {
		log.Error(err.Error())
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}
}
