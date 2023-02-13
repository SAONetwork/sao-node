package gateway

import (
	"context"
	"fmt"
	"io"
	"sao-node/store"
	"sao-node/types"
	"time"
)

type LocalGatewayProtocol struct {
	GatewayProtocolHandler
	chans        map[string]chan interface{}
	storeManager *store.StoreManager
}

func NewLocalGatewayProtocol(
	ctx context.Context,
	chans map[string]chan interface{},
	storeManager *store.StoreManager,
	handler GatewayProtocolHandler,
) LocalGatewayProtocol {
	p := LocalGatewayProtocol{
		chans:                  chans,
		storeManager:           storeManager,
		GatewayProtocolHandler: handler,
	}
	go p.listenShardComplete(ctx)
	return p
}

func (l LocalGatewayProtocol) Stop(_ context.Context) error {
	log.Info("stopping local gateway protocol ...")
	return nil
}

func (l LocalGatewayProtocol) listenShardComplete(ctx context.Context) {
	if c, exists := l.chans[types.ShardCompleteProtocol]; exists {
		for {
			select {
			case t, ok := <-c:
				if !ok {
					return
				}

				resp := l.HandleShardComplete(t.(types.ShardCompleteReq))
				if resp.Code != 0 {
					// TODO: consider how to continue this order
					log.Errorf(resp.Message)
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func (l LocalGatewayProtocol) RequestShardAssign(ctx context.Context, req types.ShardAssignReq, _ string) types.ShardAssignResp {
	l.chans[types.ShardAssignProtocol] <- req
	return types.ShardAssignResp{Code: 0}
}

func (l LocalGatewayProtocol) RequestShardLoad(ctx context.Context, req types.ShardLoadReq, _ string) types.ShardLoadResp {
	returnErr := func(code uint64, errMsg string) types.ShardLoadResp {
		return types.ShardLoadResp{
			Code:       code,
			Message:    errMsg,
			OrderId:    req.OrderId,
			Cid:        req.Cid,
			Content:    nil,
			RequestId:  req.RequestId,
			ResponseId: time.Now().UnixMilli(),
		}
	}

	reader, err := l.storeManager.Get(ctx, req.Cid)
	if err != nil {
		return returnErr(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("get cid(%v) from store manager error: %v", req.Cid, err),
		)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return returnErr(
			types.ErrorCodeInternalErr,
			fmt.Sprintf("failed to read from store manager: %v", err),
		)
	}
	return types.ShardLoadResp{
		Code:       0,
		OrderId:    req.OrderId,
		Cid:        req.Cid,
		Content:    content,
		RequestId:  req.RequestId,
		ResponseId: time.Now().UnixMilli(),
	}
}
