package gql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/graph-gophers/graphql-go"
)

type order struct {
	Id          string
	Creator     string
	Owner       string
	Provider    string
	Cid         string
	Duration    int32
	Status      string
	Replica     int32
	Amount      string
	Size        string
	Operation   string
	CreatedAt   int32
	Timeout     int32
	DataId      string
	Commit      string
	UnitPrice   string
	OrderShards []*OrderShard
}

type OrderShard struct {
	ShardId int32
	Status  int32
}

type orderList struct {
	TotalCount int32
	Orders     []*order
	More       bool
}

type OrderQueryArgs struct {
	Owner graphql.NullString
}

// query: order(orderId) Order
func (r *resolver) Order(ctx context.Context, args struct{ ID graphql.ID }) (*order, error) {
	var orderId string
	orderId = string(args.ID)

	row := r.indexSvc.Db.QueryRowContext(ctx, `SELECT id, creator, owner, provider, cid, duration, status, 
    replica, amount, size, operation, createdAt, timeout, dataId, commitId, unitPrice, shards FROM ORDERS WHERE id= ?`, orderId)

	ShardIds := ""
	ord := &order{}
	err := row.Scan(&ord.Id, &ord.Creator, &ord.Owner, &ord.Provider, &ord.Cid, &ord.Duration, &ord.Status,
		&ord.Replica, &ord.Amount, &ord.Size, &ord.Operation, &ord.CreatedAt, &ord.Timeout, &ord.DataId, &ord.Commit,
		&ord.UnitPrice, &ShardIds)
	if err != nil {
		return nil, fmt.Errorf("database scan error: %v", err)
	}

	// ShardIds is string with comma split, Convert it to uint64 array
	for _, shard := range strings.Split(ShardIds, ",") {
		shardUint64, err := strconv.ParseUint(shard, 10, 64)
		if err != nil {
			log.Errorf("failed to parse shard %s: %w", shard, err)
		}
		saoShard, err := r.chainSvc.GetShard(ctx, shardUint64)
		if err != nil {
			log.Errorf("failed to get shard %d: %w", shardUint64, err)
		}
		// put shardId and saoShard.Status into &ord.OrderShards
		ord.OrderShards = append(ord.OrderShards, &OrderShard{ShardId: int32(shardUint64), Status: saoShard.Status})
	}

	return ord, nil
}

// query: orders(Owner) OrderList
func (r *resolver) Orders(ctx context.Context, args OrderQueryArgs) (*orderList, error) {
	queryStr := `SELECT id, creator, owner, provider, cid, duration, status, replica, amount, size, operation, 
    createdAt, timeout, dataId, commitId, unitPrice FROM ORDERS`

	var params []interface{}

	if args.Owner.Set && args.Owner.Value != nil {
		queryStr += " WHERE owner = ?"
		params = append(params, *args.Owner.Value)
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, queryStr, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*order, 0)
	for rows.Next() {
		ord := &order{}
		err = rows.Scan(&ord.Id, &ord.Creator, &ord.Owner, &ord.Provider, &ord.Cid, &ord.Duration, &ord.Status,
			&ord.Replica, &ord.Amount, &ord.Size, &ord.Operation, &ord.CreatedAt, &ord.Timeout, &ord.DataId,
			&ord.Commit, &ord.UnitPrice)
		if err != nil {
			return nil, err
		}
		orders = append(orders, ord)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &orderList{
		TotalCount: int32(len(orders)),
		Orders:     orders,
		More:       false,
	}, nil
}

func (r *resolver) OrderCount(ctx context.Context) (int32, error) {
	return 0, nil
}

func (o *order) ID() graphql.ID {
	return graphql.ID(o.Id)
}
