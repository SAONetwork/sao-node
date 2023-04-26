package gql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type listingInfo struct {
	CommitId        string       `json:"CommitId"`
	DataId  string       `json:"dataId"`
	Alias      string `json:"Alias"`
	Price   string       `json:"price"`
	TokenId string       `json:"tokenId"`
	ItemDataId string `json:"itemDataId"`
	ChainType string `json:"chainType"`
	Time    types.Uint64 `json:"time"`
}

type listingInfoArgs struct {
	DataId *string
	TokenId *string
}

// query: listingInfo(dataId) ListingInfo
func (r *resolver) ListingInfo(ctx context.Context, args listingInfoArgs) (*listingInfo, error) {
	if args.DataId == nil && args.TokenId == nil {
		return nil, fmt.Errorf("Either DataId or TokenId must be provided")
	}

	var row *sql.Row

	if args.DataId != nil {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM LISTING_INFO WHERE DATAID = ?", *args.DataId)
	} else {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM LISTING_INFO WHERE TOKENID = ?", *args.TokenId)
	}
	var info listingInfo
	err := row.Scan(
		&info.CommitId,
		&info.DataId,
		&info.Alias,
		&info.Price,
		&info.TokenId,
		&info.ItemDataId,
		&info.ChainType,
		&info.Time,
	)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func (l *listingInfo) ID() graphql.ID {
	return graphql.ID(l.CommitId)
}