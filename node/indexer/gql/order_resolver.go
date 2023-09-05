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
	Owner   *graphql.NullString
	Limit   *int32
	Offset  *int32
	OrderBy *OrderByInput
}

type OrderByInput struct {
	Column OrderByColumn
	Order  SortOrder
}

type OrderByColumn string
type SortOrder string

const (
	// For OrderByColumn
	ID         OrderByColumn = "ID"
	CREATOR    OrderByColumn = "CREATOR"
	PROVIDER   OrderByColumn = "PROVIDER"
	DURATION   OrderByColumn = "DURATION"
	STATUS     OrderByColumn = "STATUS"
	REPLICA    OrderByColumn = "REPLICA"
	AMOUNT     OrderByColumn = "AMOUNT"
	SIZE       OrderByColumn = "SIZE"
	CREATED_AT OrderByColumn = "CREATED_AT"

	// For SortOrder
	ASC  SortOrder = "ASC"
	DESC SortOrder = "DESC"
)

var columnMapping = map[OrderByColumn]string{
	ID:         "id",
	CREATOR:    "creator",
	PROVIDER:   "provider",
	DURATION:   "duration",
	STATUS:     "status",
	REPLICA:    "replica",
	AMOUNT:     "amount",
	SIZE:       "size",
	CREATED_AT: "createdAt",
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
	baseQuery := `FROM ORDERS`
	queryStr := `SELECT id, creator, owner, provider, cid, duration, status, replica, amount, size, operation, 
    createdAt, timeout, dataId, commitId, unitPrice ` + baseQuery

	var params []interface{}

	if args.Owner.Set && args.Owner.Value != nil {
		queryStr += " WHERE owner = ?"
		baseQuery += " WHERE owner = ?" // needed for count query too
		params = append(params, args.Owner.Value)
	}

	// Add ordering
	if args.OrderBy != nil {
		sqlColumn, exists := columnMapping[args.OrderBy.Column]
		if !exists {
			return nil, fmt.Errorf("Invalid orderBy column provided")
		}
		queryStr += fmt.Sprintf(" ORDER BY %s %s", sqlColumn, args.OrderBy.Order)
	} else {
		queryStr += " ORDER BY CAST(id AS UNSIGNED) DESC"
	}

	// Add limit and offset if provided
	if args.Limit != nil {
		queryStr += " LIMIT ?"
		params = append(params, int(*args.Limit))
	}
	if args.Offset != nil {
		queryStr += " OFFSET ?"
		params = append(params, int(*args.Offset))
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

	// Fetch total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int32
	err = r.indexSvc.Db.QueryRowContext(ctx, countQuery, params...).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	moreResults := false
	if args.Limit != nil && len(orders) == int(*args.Limit) {
		moreResults = true
	}

	return &orderList{
		TotalCount: totalCount,
		Orders:     orders,
		More:       moreResults,
	}, nil
}

func (r *resolver) OrderCount(ctx context.Context) (int32, error) {
	return 0, nil
}

func (o *order) ID() graphql.ID {
	return graphql.ID(o.Id)
}
