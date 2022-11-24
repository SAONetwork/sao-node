package api

import (
	"context"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/types"

	"github.com/filecoin-project/go-jsonrpc/auth"
)

type GatewayApi interface {
	Test(ctx context.Context, msg string) (string, error)                                                                            //perm:none
	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error)                                                         //perm:none
	AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error)                                                            //perm:admin
	Create(ctx context.Context, orderProposal types.OrderStoreProposal, orderId uint64, content []byte) (apitypes.CreateResp, error) //perm:write
	CreateFile(ctx context.Context, orderProposal types.OrderStoreProposal, orderId uint64) (apitypes.CreateResp, error)             //perm:write
	Load(ctx context.Context, req apitypes.LoadReq) (apitypes.LoadResp, error)                                                       //perm:read
	Delete(ctx context.Context, onwer string, keyword string, group string) (apitypes.DeleteResp, error)                             //perm:write
	ShowCommits(ctx context.Context, onwer string, keyword string, group string) (apitypes.ShowCommitsResp, error)                   //perm:read
	Update(ctx context.Context, orderProposal types.OrderStoreProposal, orderId uint64, patch []byte) (apitypes.UpdateResp, error)   //perm:write
	Renew(ctx context.Context, timeout int32, renewModels map[string]uint64) error                                                   //perm:write
	GetPeerInfo(ctx context.Context) (apitypes.GetPeerInfoResp, error)                                                               //perm:read
	GenerateToken(ctx context.Context, owner string) (apitypes.GenerateTokenResp, error)                                             //perm:read
	GetHttpUrl(ctx context.Context, dataId string) (apitypes.GetUrlResp, error)                                                      //perm:read
	GetIpfsUrl(ctx context.Context, cid string) (apitypes.GetUrlResp, error)                                                         //perm:read
	NodeAddress(ctx context.Context) (string, error)                                                                                 //perm:read
	NetPeers(context.Context) ([]types.PeerInfo, error)                                                                              //perm:read
}
