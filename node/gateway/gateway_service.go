package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sao-storage-node/node/chain"
	"sao-storage-node/node/config"
	"sao-storage-node/store"
	"sao-storage-node/types"
	"sao-storage-node/utils"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
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
	Shards  map[string]*modeltypes.ShardMeta
}

type FetchResult struct {
	Cid     string
	Content []byte
}

type GatewaySvcApi interface {
	QueryMeta(ctx context.Context, account string, key string, group string, height int64) (*types.Model, error)
	CommitModel(ctx context.Context, clientProposal types.ClientOrderProposal, orderId uint64, content []byte) (*CommitResult, error)
	FetchContent(ctx context.Context, meta *types.Model) (*FetchResult, error)
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
		shardStreamHandler: NewShardStreamHandler(ctx, host, cfg.Transport.StagingPath),
		storeManager:       storeManager,
		nodeAddress:        nodeAddress,
		stagingPath:        cfg.Transport.StagingPath,
		cfg:                cfg,
	}

	return cs
}

func (gs *GatewaySvc) QueryMeta(ctx context.Context, account string, key string, group string, height int64) (*types.Model, error) {
	var res *modeltypes.QueryGetMetadataResponse = nil
	var err error
	var dataId string
	if utils.IsDataId(key) {
		dataId = key
	} else {
		dataId, err = gs.chainSvc.QueryDataId(ctx, fmt.Sprintf("%s-%s-%s", account, key, group))
		if err != nil {
			return nil, err
		}
	}
	res, err = gs.chainSvc.QueryMeta(ctx, dataId, height)
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
		DataId:  res.Metadata.DataId,
		Alias:   res.Metadata.Alias,
		GroupId: res.Metadata.GroupId,
		Owner:   res.Metadata.Owner,
		OrderId: res.Metadata.OrderId,
		Tags:    res.Metadata.Tags,
		// Cid: res.Metadata.Cid,
		Shards:   res.Shards,
		CommitId: commitInfo[0],
		Commits:  res.Metadata.Commits,
		// Content: N/a,
		ExtendInfo: res.Metadata.ExtendInfo,
	}, nil
}

func (gs *GatewaySvc) FetchContent(ctx context.Context, meta *types.Model) (*FetchResult, error) {
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
			shardContent, err = gs.shardStreamHandler.Fetch(shard.Peer, shardCid)
			if err != nil {
				return nil, err
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

	if len(content) > gs.cfg.Cache.ContentLimit {
		// large size content should go through P2P channel

		path, err := homedir.Expand(gs.cfg.Gateway.HttpFileServerPath)
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

		content = make([]byte, 0)
	}

	return &FetchResult{
		Cid:     contentCid.String(),
		Content: content,
	}, nil
}

func (gs *GatewaySvc) CommitModel(ctx context.Context, clientProposal types.ClientOrderProposal, orderId uint64, content []byte) (*CommitResult, error) {
	// TODO: consider store node may ask earlier than file split
	// TODO: if big data, consider store to staging dir.
	// TODO: support split file.
	// TODO: support marshal any content
	orderProposal := clientProposal.Proposal
	err := StageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid, content)
	if err != nil {
		return nil, err
	}

	if orderId == 0 {
		var metadata string
		if orderProposal.IsUpdate {
			metadata = fmt.Sprintf(
				`{"dataId": "%s", "commit": "%s", "update": true}`,
				orderProposal.DataId,
				orderProposal.CommitId,
			)
		} else {
			metadata = fmt.Sprintf(
				`{"alias": "%s", "dataId": "%s", "extendInfo": "%s", "groupId": "%s", "commit": "%s", "update": false}`,
				orderProposal.Alias,
				orderProposal.DataId,
				orderProposal.ExtendInfo,
				orderProposal.GroupId,
				orderProposal.CommitId,
			)
		}

		m, err := json.Marshal(clientProposal)
		if err != nil {
			return nil, err
		}
		log.Info("metadata1: ", string(m))
		log.Info("metadata2: ", metadata)

		orderId, txId, err := gs.chainSvc.StoreOrder(ctx, gs.nodeAddress, orderProposal.Owner, gs.nodeAddress, orderProposal.Cid, orderProposal.Duration, orderProposal.Replica, metadata)
		if err != nil {
			return nil, err
		}
		log.Infof("StoreOrder tx succeed. orderId=%d tx=%s", orderId, txId)
	} else {
		txId, err := gs.chainSvc.OrderReady(ctx, gs.nodeAddress, orderId)
		if err != nil {
			return nil, err
		}
		log.Infof("OrderReady tx succeed. orderId=%d tx=%s", orderId, txId)
	}

	doneChan := make(chan chain.OrderCompleteResult)
	err = gs.chainSvc.SubscribeOrderComplete(ctx, orderId, doneChan)
	if err != nil {
		return nil, err
	}

	log.Debugf("SubscribeOrderComplete")

	timeout := false
	select {
	case <-doneChan:
	case <-time.After(chain.Blocktime * time.Duration(clientProposal.Proposal.Timeout)):
		timeout = true
	case <-ctx.Done():
		timeout = true
	}
	close(doneChan)

	err = gs.chainSvc.UnsubscribeOrderComplete(ctx, orderId)
	if err != nil {
		log.Error(err)
	} else {
		log.Debugf("UnsubscribeOrderComplete")
	}

	log.Debugf("unstage shard %s/%s/%v", gs.stagingPath, orderProposal.Owner, orderProposal.Cid)
	err = UnstageShard(gs.stagingPath, orderProposal.Owner, orderProposal.Cid)
	if err != nil {
		return nil, err
	}

	if timeout {
		// TODO: timeout handling
		return nil, errors.Errorf("process order %d timeout.", orderId)
	} else {
		meta, err := gs.chainSvc.QueryMeta(ctx, orderProposal.DataId, 0)
		if err != nil {
			return nil, err
		}
		log.Debugf("order %d complete: dataId=%s", meta.Metadata.OrderId, &meta.Metadata.DataId)

		return &CommitResult{
			OrderId: meta.Metadata.OrderId,
			DataId:  meta.Metadata.DataId,
			Commit:  meta.Metadata.Commit,
			Commits: meta.Metadata.Commits,
			Shards:  meta.Shards,
			Cid:     orderProposal.Cid.String(),
		}, nil
	}
}

func (cs *GatewaySvc) Stop(ctx context.Context) error {
	log.Info("stopping order service...")
	cs.shardStreamHandler.Stop(ctx)

	return nil
}
