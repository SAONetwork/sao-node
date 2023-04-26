package gql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type fileInfo struct {
	CommitId     string
	DataID       string
	Alias        string
	CreatedAt    types.Uint64
	FileDataID   string
	ContentType  string
	Owner        string
	Filename     string
	FileCategory string
}

// query: fileInfo(id, userDataId) FileInfo
func (r *resolver) FileInfo(ctx context.Context, args struct{ ID graphql.ID; UserDataId *string }) (*fileInfo, error) {
	var commitId uuid.UUID
	err := commitId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	var fi fileInfo
	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM FILE_INFO WHERE COMMITID = ?", commitId)
	err = row.Scan(
		&fi.CommitId,
		&fi.DataID,
		&fi.Alias,
		&fi.CreatedAt,
		&fi.FileDataID,
		&fi.ContentType,
		&fi.Owner,
		&fi.Filename,
		&fi.FileCategory,
	)
	if err != nil {
		return nil, err
	}

	// Find verse by fileInfo ID
	var verse struct {
		DataID      string
		Price       float64
		FileIDs     string
	}
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT DATAID, PRICE, FILEIDS FROM VERSE WHERE FILEIDS LIKE ?", "%"+fi.CommitId+"%").Scan(&verse.DataID, &verse.Price, &verse.FileIDs)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		return &fi, nil
	}

	// If verse price is greater than 0, check if there's a PurchaseOrder record with ItemDataID = verse.DATAID and BuyerDataID = userDataId
	if verse.Price > 0 {
		var count int
		if args.UserDataId != nil {
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", verse.DataID, *args.UserDataId).Scan(&count)
		} else {
			return nil, errors.New("userDataId is required when the file is charged")
		}
		if err != nil {
			return nil, err
		}

		if count == 0 {
			return nil, errors.New("the file is charged and not paid yet")
		}
	}

	return &fi, nil
}

func (fi *fileInfo) ID() graphql.ID {
	return graphql.ID(fi.CommitId)
}
