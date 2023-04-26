package gql

import (
	"context"
	"database/sql"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type purchaseOrderList struct {
	TotalCount     int32           `json:"totalCount"`
	PurchaseOrders []*purchaseOrder `json:"purchaseOrders"`
	More           bool            `json:"more"`
}

type purchaseOrder struct {
	CommitId        string       `json:"CommitId"`
	DataId  string       `json:"dataId"`
	Alias      string `json:"Alias"`
	OrderID     types.Uint64 `json:"orderId"`
	ItemDataID  string       `json:"itemDataId"`
	BuyerDataID string       `json:"buyerDataId"`
	OrderTxHash string       `json:"orderTxHash"`
	ChainType   string       `json:"chainType"`
	Price       string       `json:"price"`
	Time        types.Uint64 `json:"time"`
	Type        int32        `json:"type"`
	ExpireTime  types.Uint64       `json:"expireTime"`
}

type purchaseOrderArgs struct {
	ItemDataID *string
}

// query: purchaseOrders(query) PurchaseOrderList
func (r *resolver) PurchaseOrders(ctx context.Context, args purchaseOrderArgs) (*purchaseOrderList, error) {
	var rows *sql.Rows
	var err error

	if args.ItemDataID != nil {
		rows, err = r.indexSvc.Db.QueryContext(ctx, "SELECT * FROM PURCHASE_ORDER WHERE ITEMDATAID = ?", *args.ItemDataID)
	} else {
		rows, err = r.indexSvc.Db.QueryContext(ctx, "SELECT * FROM PURCHASE_ORDER")
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*purchaseOrder

	for rows.Next() {
		var order purchaseOrder
		err := rows.Scan(
			&order.CommitId,
			&order.DataId,
			&order.Alias,
			&order.OrderID,
			&order.ItemDataID,
			&order.BuyerDataID,
			&order.OrderTxHash,
			&order.ChainType,
			&order.Price,
			&order.Time,
			&order.Type,
			&order.ExpireTime,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	var totalCount int32
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER").Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	more := false

	return &purchaseOrderList{
		TotalCount:     totalCount,
		PurchaseOrders: orders,
		More:           more,
	}, nil
}

func (l *purchaseOrder) ID() graphql.ID {
	return graphql.ID(l.CommitId)
}