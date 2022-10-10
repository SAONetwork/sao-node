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

func (sc SaoClient) Create(ctx context.Context, orderMeta types.OrderMeta, content string) (apitypes.CreateResp, error) {
	return sc.gatewayApi.Create(ctx, orderMeta, content)
}
