package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sao-node/types"
	"time"

	"github.com/mitchellh/go-homedir"
)

type LocalStorageProtocol struct {
	StorageProtocolHandler
	chans       map[string]chan interface{}
	stagingPath string
}

func NewLocalStorageProtocol(
	ctx context.Context,
	chans map[string]chan interface{},
	stagingPath string,
	handler StorageProtocolHandler,
) LocalStorageProtocol {
	p := LocalStorageProtocol{
		chans:                  chans,
		stagingPath:            stagingPath,
		StorageProtocolHandler: handler,
	}
	go p.listenShardAssign(ctx)
	return p
}

func (l LocalStorageProtocol) Stop(_ context.Context) error {
	log.Info("stopping local storage protocol")
	return nil
}

func (l LocalStorageProtocol) listenShardAssign(ctx context.Context) {
	for {
		select {
		case t, ok := <-l.chans[types.ShardAssignProtocol]:
			if !ok {
				return
			}
			// process
			resp := l.HandleShardAssign(t.(types.ShardAssignReq))
			if resp.Code != 0 {
				log.Errorf(resp.Message)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (l LocalStorageProtocol) RequestShardComplete(ctx context.Context, req types.ShardCompleteReq, _ string) types.ShardCompleteResp {
	l.chans[types.ShardCompleteProtocol] <- req
	return types.ShardCompleteResp{Code: 0}
}

func (l LocalStorageProtocol) RequestShardStore(ctx context.Context, req types.ShardLoadReq, _ string) types.ShardLoadResp {
	resp := types.ShardLoadResp{
		OrderId:   req.OrderId,
		Cid:       req.Cid,
		RequestId: req.RequestId,
	}

	path, err := homedir.Expand(l.stagingPath)
	if err != nil {
		resp.ResponseId = time.Now().UnixMilli()
		resp.Code = types.ErrorCodeInternalErr
		resp.Message = fmt.Sprintf("invalid path: %s", l.stagingPath)
		return resp
	}

	filename := filepath.Join(path, req.Owner, req.Cid.String())
	bytes, err := os.ReadFile(filename)
	if err != nil {
		resp.ResponseId = time.Now().UnixMilli()
		resp.Code = types.ErrorCodeInternalErr
		resp.Message = fmt.Sprintf("read file failed: %s", filename)
		return resp
	} else {
		resp.Content = bytes
		resp.Code = 0
		resp.ResponseId = time.Now().UnixMilli()
		return resp
	}
}
