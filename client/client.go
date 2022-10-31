package client

import (
	"context"
	"sao-storage-node/api"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/types"
)

type SaoClient struct {
	gatewayApi api.GatewayApi
}

func NewSaoClient(api api.GatewayApi) SaoClient {
	return SaoClient{
		gatewayApi: api,
	}
}

func (sc SaoClient) Test(ctx context.Context) (string, error) {
	resp, err := sc.gatewayApi.Test(ctx, "hello")
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (sc SaoClient) Create(ctx context.Context, orderMeta types.OrderMeta, content []byte) (apitypes.CreateResp, error) {
	return sc.gatewayApi.Create(ctx, orderMeta, content)
}

func (sc SaoClient) CreateFile(ctx context.Context, orderMeta types.OrderMeta) (apitypes.CreateResp, error) {
	return sc.gatewayApi.CreateFile(ctx, orderMeta)
}

func (sc SaoClient) Load(ctx context.Context, owner string, alias string) (apitypes.LoadResp, error) {
	return sc.gatewayApi.Load(ctx, owner, alias)
}

func (sc SaoClient) Delete(ctx context.Context, owner string, alias string) (apitypes.DeleteResp, error) {
	return sc.gatewayApi.Delete(ctx, owner, alias)
}

func (sc SaoClient) GetPeerInfo(ctx context.Context) (apitypes.GetPeerInfoResp, error) {
	return sc.gatewayApi.GetPeerInfo(ctx)
}
