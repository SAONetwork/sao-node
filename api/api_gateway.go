package api

import (
	"context"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/types"
)

type GatewayApi interface {
	Test(ctx context.Context, msg string) (string, error)
	Create(ctx context.Context, orderProposal types.ClientOrderProposal, orderMeta types.OrderMeta, content []byte) (apitypes.CreateResp, error)
	CreateFile(ctx context.Context, orderMeta types.OrderMeta) (apitypes.CreateResp, error)
	Load(ctx context.Context, orderMeta types.OrderMeta) (apitypes.LoadResp, error)
	Delete(ctx context.Context, onwer string, key string, group string) (apitypes.DeleteResp, error)
	ShowCommits(ctx context.Context, onwer string, key string, group string) (apitypes.ShowCommitsResp, error)
	Update(ctx context.Context, orderProposal types.ClientOrderProposal, orderMeta types.OrderMeta, patch []byte) (apitypes.UpdateResp, error)
	GetPeerInfo(ctx context.Context) (apitypes.GetPeerInfoResp, error)
	GenerateToken(ctx context.Context, owner string) (apitypes.GenerateTokenResp, error)
	GetHttpUrl(ctx context.Context, dataId string) (apitypes.GetUrlResp, error)
	GetIpfsUrl(ctx context.Context, cid string) (apitypes.GetUrlResp, error)
	NodeAddress(ctx context.Context) (string, error)
}
