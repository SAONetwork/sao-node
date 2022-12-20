package api

import (
	"context"
	apitypes "sao-node/api/types"
	"sao-node/types"

	"github.com/filecoin-project/go-jsonrpc/auth"
)

type SaoApi interface {
	// MethodGroup: Auth

	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) //perm:none
	AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error)    //perm:admin

	// MethodGroup: Model
	// The Model method group contains methods for manipulating data models.

	// ModelCreateFile create data model as a file
	ModelCreateFile(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64) (apitypes.CreateResp, error) //perm:write
	// ModelCreate create a normal data model
	ModelCreate(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, content []byte) (apitypes.CreateResp, error) //perm:write
	// ModelLoad load an existing data model
	ModelLoad(ctx context.Context, req *types.MetadataProposal) (apitypes.LoadResp, error) //perm:read
	// ModelDelete delete an existing model
	ModelDelete(ctx context.Context, req *types.OrderTerminateProposal) (apitypes.DeleteResp, error) //perm:write
	// ModelShowCommits list a data models' historical commits
	ModelShowCommits(ctx context.Context, req *types.MetadataProposal) (apitypes.ShowCommitsResp, error) //perm:read
	// ModelUpdate update an existing data model
	ModelUpdate(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, patch []byte) (apitypes.UpdateResp, error) //perm:write

	// MethodGroup: Common

	// GetPeerInfo get current node's peer information
	GetPeerInfo(ctx context.Context) (apitypes.GetPeerInfoResp, error) //perm:read
	// GenerateToken
	GenerateToken(ctx context.Context, owner string) (apitypes.GenerateTokenResp, error) //perm:read
	// GetHttpUrl
	GetHttpUrl(ctx context.Context, dataId string) (apitypes.GetUrlResp, error) //perm:read
	// GetIpfsUrl
	GetIpfsUrl(ctx context.Context, cid string) (apitypes.GetUrlResp, error) //perm:read
	// GetNodeAddress get current node's sao chain address
	GetNodeAddress(ctx context.Context) (string, error) //perm:read
	// GetNetPeers get current node's connected peer list
	GetNetPeers(context.Context) ([]types.PeerInfo, error) //perm:read
}
