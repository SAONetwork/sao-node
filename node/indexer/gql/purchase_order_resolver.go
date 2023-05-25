package gql

import (
	"context"
	"database/sql"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type purchaseOrderList struct {
	TotalCount     int32            `json:"totalCount"`
	PurchaseOrders []*purchaseOrder `json:"purchaseOrders"`
}

type purchaseOrder struct {
	CommitId    string       `json:"CommitId"`
	DataId      string       `json:"dataId"`
	Alias       string       `json:"Alias"`
	OrderID     types.Uint64 `json:"orderId"`
	ItemDataID  string       `json:"itemDataId"`
	BuyerDataID string       `json:"buyerDataId"`
	OrderTxHash string       `json:"orderTxHash"`
	ChainType   string       `json:"chainType"`
	Price       string       `json:"price"`
	Time        types.Uint64 `json:"time"`
	Type        int32        `json:"type"`
	ExpireTime  types.Uint64 `json:"expireTime"`
	VerseDigest string       `json:"verseDigest"`
	UserName    string       `json:"userName"`
	Avatar      string       `json:"avatar"`
}

type purchaseOrderArgs struct {
	ItemDataId *string
	UserDataId *string
	Limit      *int32
	Offset     *int32
}

type totalEarningsArgs struct {
	UserDataId string
}

type earningsByMonth struct {
	Month  string
	Total  float64
}

// query: purchaseOrders(query) PurchaseOrderList
func (r *resolver) PurchaseOrders(ctx context.Context, args purchaseOrderArgs) (*purchaseOrderList, error) {
	var rows *sql.Rows
	var err error
	limit := 10
	offset := 0

	if args.Limit != nil {
		limit = int(*args.Limit)
	}
	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	if args.UserDataId != nil {
		rows, err = r.indexSvc.Db.QueryContext(ctx, `
			SELECT * 
			FROM PURCHASE_ORDER 
			WHERE ((TYPE = 2 AND ITEMDATAID = ?) OR 
			(TYPE = 1 AND ITEMDATAID IN (SELECT DATAID FROM VERSE WHERE OWNER = ?))) 
			LIMIT ? OFFSET ?`,
			*args.UserDataId, *args.UserDataId, limit, offset)
	} else if args.ItemDataId != nil {
		rows, err = r.indexSvc.Db.QueryContext(ctx, "SELECT * FROM PURCHASE_ORDER WHERE ITEMDATAID = ? LIMIT ? OFFSET ?", *args.ItemDataId, limit, offset)
	} else {
		rows, err = r.indexSvc.Db.QueryContext(ctx, "SELECT * FROM PURCHASE_ORDER LIMIT ? OFFSET ?", limit, offset)
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

		// Fetching avatar and username from the user_profile table
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT AVATAR, USERNAME FROM USER_PROFILE WHERE DATAID = ?", order.BuyerDataID).Scan(&order.Avatar, &order.UserName)
		if err != nil {
			return nil, err
		}

		// If order type is 1, fetch owner and digest from the verse table
		if order.Type == 1 {
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT OWNER, DIGEST FROM VERSE WHERE DATAID = ?", order.ItemDataID).Scan(&order.UserName, &order.VerseDigest)
			if err != nil {
				return nil, err
			}
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

	return &purchaseOrderList{
		TotalCount:     totalCount,
		PurchaseOrders: orders,
	}, nil
}

func (r *resolver) EarningsByMonth(ctx context.Context, args totalEarningsArgs) ([]*earningsByMonth, error) {
	rows, err := r.indexSvc.Db.QueryContext(ctx, `
        SELECT 
            strftime('%Y-%m', datetime(TIME, 'unixepoch')) AS Month, 
            SUM(CAST(PRICE as REAL)) AS Total 
        FROM PURCHASE_ORDER
        WHERE ((TYPE = 2 AND ITEMDATAID = ?) OR (TYPE = 1 AND ITEMDATAID IN (SELECT DATAID FROM VERSE WHERE OWNER = ?)))
            AND TIME >= strftime('%s', date('now', '-6 months'))
        GROUP BY Month
        ORDER BY Month DESC`,
		args.UserDataId, args.UserDataId)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var earnings []*earningsByMonth

	for rows.Next() {
		var e earningsByMonth
		err := rows.Scan(&e.Month, &e.Total)
		if err != nil {
			return nil, err
		}
		earnings = append(earnings, &e)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return earnings, nil
}

func (l *purchaseOrder) ID() graphql.ID {
	return graphql.ID(l.CommitId)
}
