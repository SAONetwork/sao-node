package gql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"io/ioutil"
	"sao-node/node/indexer/gql/types"
	"strconv"
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

type fileContent struct {
	CommitId    string
	DataID      string
	Alias       string
	CreatedAt   types.Uint64
	Owner       string
	ContentPath string
}

// query: fileInfo(id, userDataId) FileInfo
func (r *resolver) FileInfo(ctx context.Context, args struct {
	ID         graphql.ID
	UserDataId *string
}) (*fileInfo, error) {
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
	var v verse
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE FILEIDS LIKE ?", "%"+fi.CommitId+"%").Scan(
		&v.CommitId,
		&v.DataId,
		&v.Alias,
		&v.CreatedAt,
		&v.FileIDs,
		&v.Owner,
		&v.Price,
		&v.Digest,
		&v.Scope,
		&v.Status,
		&v.NftTokenID,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		return &fi, nil
	}

	if args.UserDataId != nil {
		// Process verse scope
		_, err = processVerseScope(ctx, r.indexSvc.Db, &v, *args.UserDataId)
		if err != nil {
			return nil, err
		}

		// If verse price is greater than 0, check if there's a PurchaseOrder record with ItemDataID = verse.DATAID and BuyerDataID = userDataId
		if v.Price != "" {
			price, err := strconv.ParseFloat(v.Price, 64)
			if err != nil {
				return nil, err
			}

			if price > 0 {
				var count int
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
				if err != nil {
					return nil, err
				}

				if count == 0 {
					return nil, errors.New("the file is charged and not paid yet")
				}
			}
		}
	}

	return &fi, nil
}

// query: file(id, userDataId) String
func (r *resolver) File(ctx context.Context, args struct {
	ID         graphql.ID
	UserDataId *string
}) (*string, error) {
	var commitId uuid.UUID
	err := commitId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	var fc fileContent
	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM FILE_CONTENT WHERE COMMITID = ?", commitId)
	err = row.Scan(
		&fc.CommitId,
		&fc.DataID,
		&fc.Alias,
		&fc.CreatedAt,
		&fc.Owner,
		&fc.ContentPath,
	)
	if err != nil {
		return nil, err
	}

	// permission check logic
	if args.UserDataId == nil || *args.UserDataId != fc.Owner {
		//return nil, errors.New("access denied")
	}

	fileContentBytes, err := ioutil.ReadFile(fc.ContentPath)
	if err != nil {
		return nil, err
	}
	fileContentString := string(fileContentBytes)
	return &fileContentString, nil
}

func (fi *fileInfo) ID() graphql.ID {
	return graphql.ID(fi.CommitId)
}
