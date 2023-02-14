package storage

import (
	"context"
	"sao-node/types"
)

type StorageProtocol interface {
	RequestShardComplete(ctx context.Context, req types.ShardCompleteReq, peer string) types.ShardCompleteResp
	RequestShardStore(ctx context.Context, req types.ShardLoadReq, peer string) types.ShardLoadResp
	RequestShardMigrate(ctx context.Context, req types.ShardMigrateReq, peer string) types.ShardMigrateResp
	Stop(ctx context.Context) error
}

type StorageProtocolHandler interface {
	HandleShardAssign(req types.ShardAssignReq) types.ShardAssignResp
	HandleShardLoad(req types.ShardLoadReq) types.ShardLoadResp
	HandleShardMigrate(req types.ShardMigrateReq) types.ShardMigrateResp
}
