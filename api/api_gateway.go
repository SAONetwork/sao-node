package api

import (
	"context"

	sidtypes "github.com/SaoNetwork/sao/x/did/types"

	apitypes "github.com/SaoNetwork/sao-node/api/types"
	"github.com/SaoNetwork/sao-node/types"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/ipfs/go-cid"
)

type SaoApi interface {
	// MethodGroup: Auth

	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) //perm:none
	AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error)    //perm:admin

	// MethodGroup: Order Job
	OrderStatus(ctx context.Context, id string) (types.OrderInfo, error) //perm:read
	OrderList(ctx context.Context) ([]types.OrderInfo, error)            //perm:read

	// MethodGroup: Shard Job
	ShardStatus(ctx context.Context, orderId uint64, cid cid.Cid) (types.ShardInfo, error) //perm:read
	ShardList(ctx context.Context) ([]types.ShardInfo, error)                              //perm:read

	// MethodGroup: Migration Job
	MigrateJobList(ctx context.Context) ([]types.MigrateInfo, error) //perm:read

	// MethodGroup: Model
	// The Model method group contains methods for manipulating data models.

	// ModelCreateFile create data model as a file
	ModelCreateFile(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64) (apitypes.CreateResp, error) //perm:write
	// ModelCreate create a normal data model
	ModelCreate(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, content []byte) (apitypes.CreateResp, error) //perm:write
	// ModelLoad load an existing data model
	ModelLoad(ctx context.Context, req *types.MetadataProposal) (apitypes.LoadResp, error) //perm:read
	// ModelDelete delete an existing model
	ModelDelete(ctx context.Context, req *types.OrderTerminateProposal, isPublish bool) (apitypes.DeleteResp, error) //perm:write
	// ModelShowCommits list a data models' historical commits
	ModelShowCommits(ctx context.Context, req *types.MetadataProposal) (apitypes.ShowCommitsResp, error) //perm:read
	// ModelUpdate update an existing data model
	ModelUpdate(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, patch []byte) (apitypes.UpdateResp, error) //perm:write
	// ModelRenewOrder renew a list of orders
	ModelRenewOrder(ctx context.Context, req *types.OrderRenewProposal, isPublish bool) (apitypes.RenewResp, error) //perm:write
	// ModelUpdatePermission update an existing model's read/write permission
	ModelUpdatePermission(ctx context.Context, req *types.PermissionProposal, isPublish bool) (apitypes.UpdatePermissionResp, error) //perm:write
	ModelMigrate(ctx context.Context, dataIds []string) (apitypes.MigrateResp, error)                                                // perm:write

	// Raise Storage Faults
	FaultsCheck(ctx context.Context, dataIds []string) (*apitypes.FileFaultsReportResp, error) // perm:write
	// Requst Check for Recoverable Storage Faults
	RecoverCheck(ctx context.Context, provider string, faultIds []string) (*apitypes.FileRecoverReportResp, error) // perm:write

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

	// DidBindingProof binding account id to sid
	DidBindingProof(ctx context.Context, rootDocId string, keys []*sidtypes.PubKey, accAuth *sidtypes.AccountAuth, proof *sidtypes.BindingProof) (string, error) //perm:write

}
