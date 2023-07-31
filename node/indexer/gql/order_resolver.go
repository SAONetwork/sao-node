package gql

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
)

type order struct {
	Id         string
	Creator    string
	Owner      string
	Provider   string
	Cid        string
	Duration   int32
	Status     string
	Replica    int32
	Amount     string
	Size       string
	Operation  string
	CreatedAt  int32
	Timeout    int32
	DataId     string
	Commit     string
	UnitPrice  string
}

type orderList struct {
	TotalCount int32
	Orders     []*order
	More       bool
}

type OrderQueryArgs struct {
	Owner  graphql.NullString
}

// query: order(orderId) Order
func (r *resolver) Order(ctx context.Context, args struct{ ID graphql.ID }) (*order, error) {
	var orderId string
	orderId = string(args.ID)

	row := r.indexSvc.Db.QueryRowContext(ctx, `SELECT id, creator, owner, provider, cid, duration, status, 
    replica, amount, size, operation, createdAt, timeout, dataId, commitId, unitPrice FROM ORDERS WHERE id= ?`, orderId)

	ord := &order{}
	err := row.Scan(&ord.Id, &ord.Creator, &ord.Owner, &ord.Provider, &ord.Cid, &ord.Duration, &ord.Status,
		&ord.Replica, &ord.Amount, &ord.Size, &ord.Operation, &ord.CreatedAt, &ord.Timeout, &ord.DataId, &ord.Commit,
		&ord.UnitPrice)
	if err != nil {
		return nil, fmt.Errorf("database scan error: %v", err)
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
