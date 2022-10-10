package node

import (
	"context"
	"sao-storage-node/node/config"
	"sao-storage-node/node/model"

	logging "github.com/ipfs/go-log/v2"
	"github.com/tendermint/tendermint/rpc/client/http"
)

var log = logging.Logger("node")

type StorageNode struct {
	ctx             context.Context
	chainHttpClient *http.HTTP
	cfg             *config.StorageNode
	manager         *model.ModelManager
}

func NewStorageNode(ctx context.Context, cfg *config.StorageNode) StorageNode {
	return StorageNode{
		ctx:     ctx,
		cfg:     cfg,
		manager: model.NewModelManager(&cfg.Cache),
	}
}

func (n *StorageNode) Start() error {
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

	//out, err := http.Subscribe(n.ctx, "", "node-login.creator='cosmos1angsar60505jnztcjxycwpmunsn5j7wl4f6rl3'")
	//if err != nil {
	//	return err
	//}
	//for o := range out {
	//	log.Infof("o: %v", o)
	//}
	return nil
}

func (n *StorageNode) Stop() error {
	var err error
	// stop tendermint http
	if n.chainHttpClient != nil {
		err = n.chainHttpClient.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}
