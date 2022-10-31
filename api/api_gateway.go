package api

import (
	"context"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/types"
)

type GatewayApi interface {
	Test(ctx context.Context, msg string) (string, error)
	Create(ctx context.Context, orderMeta types.OrderMeta, content []byte) (apitypes.CreateResp, error)
	CreateFile(ctx context.Context, orderMeta types.OrderMeta) (apitypes.CreateResp, error)
	Load(ctx context.Context, onwer string, alias string) (apitypes.LoadResp, error)
	Delete(ctx context.Context, onwer string, alias string) (apitypes.DeleteResp, error)
	NodeAddress(ctx context.Context) (string, error)
}
