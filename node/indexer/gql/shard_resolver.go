package gql

import (
	"context"
	"fmt"

	"github.com/SaoNetwork/sao-node/node/indexer/gql/types"
	"github.com/SaoNetwork/sao-node/utils"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
)

type shard struct {
	ShardId types.Uint64
	OrderId types.Uint64
	Sp      string
	Cid     string
}

type shardList struct {
	TotalCount int32
	Shards     []*shard
	More       bool
}

// query: shard(id) Shard
func (r *resolver) Shard(ctx context.Context, args struct{ ID graphql.ID }) (*shard, error) {
	var shardId uuid.UUID
	err := shardId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT SHARDID, ORDERID, SP, CID WHERE COMMITID="+shardId.String())
	var ShardId types.Uint64
	var OrderId types.Uint64
	var Sp string
	var Cid string
	err = row.Scan(&ShardId, &OrderId, &Sp, &Cid)
	if err != nil {
		return nil, err
	}

	return &shard{
		ShardId, OrderId, Sp, Cid,
	}, nil
}

// query: shards(cursor, offset, limit) ShardList
func (r *resolver) Shards(ctx context.Context, args struct{ Query graphql.NullString }) (*shardList, error) {
	queryStr := "SELECT SHARDID, ORDERID, SP, CID WRITER FROM SP_SHARD "
	if args.Query.Set && args.Query.Value != nil {
		queryStr = queryStr + *args.Query.Value
	}
	rows, err := r.indexSvc.Db.QueryContext(ctx, queryStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shards := make([]*shard, 0)
	for rows.Next() {
		var ShardId types.Uint64
		var OrderId types.Uint64
		var Sp string
		var Cid string
		err = rows.Scan(&ShardId, &OrderId, &Sp, &Cid)
		if err != nil {
			return nil, err
		}
		shards = append(shards, &shard{
			ShardId, OrderId, Sp, Cid,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &shardList{
		TotalCount: int32(len(shards)),
		Shards:     shards,
		More:       false,
	}, nil
}

func (r *resolver) ShardCount(ctx context.Context) (int32, error) {
	return 0, nil
}

func (s *shard) ID() graphql.ID {
	return graphql.ID(utils.GenerateDataId(fmt.Sprintf("%d", s.ShardId)))
}
