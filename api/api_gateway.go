package api

import (
	"context"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/types"
)

type GatewayApi interface {
	Test(ctx context.Context, msg string) (string, error)
	Create(ctx context.Context, orderMeta types.OrderMeta, commit any) (apitypes.CreateResp, error)
	NodeAddress(ctx context.Context) (string, error)
}
