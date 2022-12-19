package gateway

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sao-node/chain"
	"sao-node/node/config"
	"sao-node/node/transport"
	"sao-node/store"
	"sao-node/types"
	"sao-node/utils"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"golang.org/x/xerrors"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
)

var log = logging.Logger("gateway")

type CommitResult struct {
	OrderId uint64
	DataId  string
	Commit  string
	Commits []string
	Cid     string
	Shards  map[string]*saotypes.ShardMeta
}

type FetchResult struct {
	Cid     string
	Content []byte
}

type GatewaySvcApi interface {
	QueryMeta(ctx context.Context, req *types.MetadataProposal, height int64) (*types.Model, error)
	CommitModel(ctx context.Context, clientProposal *types.OrderStoreProposal, orderId uint64, content []byte) (*CommitResult, error)
	FetchContent(ctx context.Context, req *types.MetadataProposal, meta *types.Model) (*FetchResult, error)
	Stop(ctx context.Context) error
}

type GatewaySvc struct {
	ctx                context.Context
	chainSvc           *chain.ChainSvc
	shardStreamHandler *ShardStreamHandler
	storeManager       *store.StoreManager
	nodeAddress        string
	stagingPath        string
	cfg                *config.Node
}

func NewGatewaySvc(ctx context.Context, nodeAddress string, chainSvc *chain.ChainSvc, host host.Host, cfg *config.Node, storeManager *store.StoreManager) *GatewaySvc {
	cs := &GatewaySvc{
		ctx:                ctx,
		chainSvc:           chainSvc,
		shardStreamHandler: NewShardStreamHandler(ctx, host, cfg.Transport.StagingPath, chainSvc, nodeAddress),
		storeManager:       storeManager,
		nodeAddress:        nodeAddress,
		stagingPath:        cfg.Transport.StagingPath,
		cfg:                cfg,
	}

	return cs
}

func (gs *GatewaySvc) QueryMeta(ctx context.Context, req *types.MetadataProposal, height int64) (*types.Model, error) {
	res, err := gs.chainSvc.QueryMetadata(ctx, req, height)
	if err != nil {
		return nil, err
	}

	log.Debugf("QueryMeta succeed. meta=%v", res.Metadata)

	commit := res.Metadata.Commits[len(res.Metadata.Commits)-1]
	commitInfo := strings.Split(commit, "\032")
	if len(commitInfo) != 2 || len(commitInfo[1]) == 0 {
		return nil, xerrors.Errorf("invalid commit information: %s", commit)
	}

	return &types.Model{
		DataId:   res.Metadata.DataId,
		Alias:    res.Metadata.Alias,
		GroupId:  res.Metadata.GroupId,
		Owner:    res.Metadata.Owner,
		OrderId:  res.Metadata.OrderId,
		Tags:     res.Metadata.Tags,
		Cid:      res.Metadata.Cid,
		Shards:   res.Shards,
		CommitId: commitInfo[0],
		Commits:  res.Metadata.Commits,
		// Content: N/a,
		ExtendInfo: res.Metadata.ExtendInfo,
	}, nil
}

func (gs *GatewaySvc) FetchContent(ctx context.Context, req *types.MetadataProposal, meta *types.Model) (*FetchResult, error) {
	contentList := make([][]byte, len(meta.Shards))
	for key, shard := range meta.Shards {
		if contentList[shard.ShardId] != nil {
			continue
		}

		shardCid, err := cid.Decode(shard.Cid)
		if err != nil {
			return nil, err
		}

		var shardContent []byte
		if key == gs.nodeAddress {
			// local shard
			if gs.storeManager == nil {
				return nil, xerrors.Errorf("local store manager not found")
			}
			reader, err := gs.storeManager.Get(ctx, shardCid)
			if err != nil {
				return nil, err
			}
			shardContent, err = io.ReadAll(reader)
			if err != nil {
				return nil, err
			}
		} else {
			// remote shard
			for _, peerInfo := range strings.Split(shard.Peer, ",") {

				_, err := multiaddr.NewMultiaddr(peerInfo)

				if err != nil {
					return nil, err
				}

				if strings.Contains(peerInfo, "udp") || strings.Contains(peerInfo, "127.0.0.1") {
					continue
				}

				shardContent, err = gs.shardStreamHandler.Fetch(req, peerInfo, shardCid)
				if err != nil {
					return nil, err
				}
				break
			}
		}
		contentList[shard.ShardId] = shardContent
	}

	var content []byte
	for _, c := range contentList {
		content = append(content, c...)
	}

	contentCid, err := utils.CalculateCid(content)
	if err != nil {
		return nil, err
	}
	if contentCid.String() != meta.Cid {
		log.Errorf("cid mismatch, expected %s, but got %s", meta.Cid, contentCid.String())
	}

	match, err := regexp.Match("^"+types.Type_Prefix_File, []byte(meta.Alias))
	if err != nil {
		return nil, err
	}

	if len(content) > gs.cfg.Cache.ContentLimit || match {
		// large size content should go through P2P channel

		path, err := homedir.Expand(gs.cfg.SaoHttpFileServer.HttpFileServerPath)
		if err != nil {
			return nil, err
		}

		file, err := os.Create(filepath.Join(path, meta.DataId))
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}

		_, err = file.Write([]byte(content))
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}

		if gs.cfg.SaoIpfs.Enable {
			_, err = gs.storeManager.Store(ctx, contentCid, bytes.NewReader(content))
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}
		}

		if len(content) > gs.cfg.Cache.ContentLimit {
			content = make([]byte, 0)
		}
	}

	return &FetchResult{
		Cid:     contentCid.String(),
		Content: content,
	}, nil
}

func (gs *GatewaySvc) CommitModel(ctx context.Context, clientProposal *types.OrderStoreProposal, orderId uint64, content []byte) (*CommitResult, error) {
	// TODO: consider store node may ask earlier than file split
	// TODO: if big data, consider store to staging dir.
	// TODO: support split file.
	// TODO: support marshal any content
	orderProposal := clientProposal.Proposal
	err := StageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid, content)
	if err != nil {
		return nil, err
	}

	var txId string
	var shards map[string]*saotypes.ShardMeta = nil
	if orderId == 0 {
		resp, txId, err := gs.chainSvc.StoreOrder(ctx, gs.nodeAddress, clientProposal)
		if err != nil {
			return nil, err
		}
		orderId = resp.OrderId
		shards = resp.Shards
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderId, txId)
	} else {
		log.Debugf("Sending OrderReady... orderId=%d", orderId)
		txId, err = gs.chainSvc.OrderReady(ctx, gs.nodeAddress, orderId)
		if err != nil {
			return nil, err
		}
		log.Infof("OrderReady tx succeed. orderId=%d tx=%s", orderId, txId)
	}

	log.Infof("assigning shards to nodes...")
	// assign shards to storage nodes

	for node, shard := range shards {
		peerInfo := ""
		log.Info("node:", node)
		log.Info("shard:", shard)
		for _, peer := range strings.Split(shard.Peer, ",") {
			log.Info("peer:", peer)
			log.Info("peerInfo:", peerInfo)
			if strings.Contains(peer, "tcp") && !strings.Contains(peer, "127.0.0.1") {
				peerInfo = peer
				break
			}
		}
		if peerInfo == "" {
			log.Errorf("no valid libp2p address found in %s", shard.Peer)
		}

		resp := types.ShardAssignResp{}
		err = transport.HandleRequest(
			ctx,
			peerInfo,
			gs.shardStreamHandler.host,
			types.ShardAssignProtocol,
			&types.ShardAssignReq{
				OrderId:  orderId,
				TxHash:   txId,
				Assignee: node,
			},
			&resp,
		)
		if err != nil {
			log.Errorf("assign order %d shards to node %s failed: %v", orderId, node, err)
		}
		if resp.Code == 0 {
			log.Infof("assigned order %d shard to node %s.", orderId, node)
		} else {
			log.Errorf("assigned order %d shards to node %s failed: %v", orderId, node, resp.Message)
		}
	}

	// TODO: wsevent
	//doneChan := make(chan chain.OrderCompleteResult)
	//err = gs.chainSvc.SubscribeOrderComplete(ctx, orderId, doneChan)
	//if err != nil {
	//	return nil, err
	//}

	log.Infof("waiting for all nodes order %d shard completion.", orderId)
	doneChan := make(chan struct{})
	gs.shardStreamHandler.AddCompleteChannel(orderId, doneChan)

	timeout := false
	select {
	case <-doneChan:
		log.Debugf("complete channel done. order %d completes", orderId)
	case <-time.After(chain.Blocktime * time.Duration(clientProposal.Proposal.Timeout)):
		timeout = true
	case <-ctx.Done():
		timeout = true
	}
	close(doneChan)

	// TODO: wsevent
	//err = gs.chainSvc.UnsubscribeOrderComplete(ctx, orderId)
	//if err != nil {
	//	log.Error(err)
	//} else {
	//	log.Debugf("UnsubscribeOrderComplete")
	//}

	log.Debugf("unstage shard %s/%s/%v", gs.stagingPath, orderProposal.Owner, orderProposal.Cid)
	err = UnstageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid)
	if err != nil {
		return nil, err
	}

	if timeout {
		// TODO: timeout handling
		return nil, errors.Errorf("process order %d timeout.", orderId)
	} else {
		order, err := gs.chainSvc.GetOrder(ctx, orderId)
		if err != nil {
			return nil, err
		}
		log.Debugf("order %d complete: dataId=%s", orderId, order.Metadata.DataId)

		shards := make(map[string]*saotypes.ShardMeta, 0)
		for peer, shard := range order.Shards {
			meta := saotypes.ShardMeta{
				ShardId:  shard.Id,
				Peer:     peer,
				Cid:      shard.Cid,
				Provider: order.Provider,
			}
			shards[peer] = &meta
		}

		return &CommitResult{
			OrderId: order.Metadata.OrderId,
			DataId:  order.Metadata.DataId,
			Commit:  order.Metadata.Commit,
			Commits: order.Metadata.Commits,
			Shards:  shards,
			Cid:     orderProposal.Cid,
		}, nil
	}
}

func (cs *GatewaySvc) Stop(ctx context.Context) error {
	log.Info("stopping order service...")
	cs.shardStreamHandler.Stop(ctx)

	return nil
}
