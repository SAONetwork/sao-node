package gql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
	"strconv"
	"strings"
)

type verse struct {
	CommitId   string `json:"CommitId"`
	DataId     string `json:"DataId"`
	Alias      string `json:"Alias"`
	CreatedAt  types.Uint64
	FileIDs    string
	Owner      string
	Price      string
	Digest     string
	Scope      string
	Status     string
	NftTokenID string
	IsPaid     bool
}

type VerseArgs struct {
	ID        *graphql.ID
	UserDataId *string
	Owner     *string
	Price     *string
	CreatedAt *types.Uint64
	Status    *string
	NftTokenId *string
}

// query: verse(id) Verse
func (r *resolver) Verse(ctx context.Context, args VerseArgs) (*verse, error) {
	var row *sql.Row
	if args.ID != nil {
		var dataId uuid.UUID
		err := dataId.UnmarshalText([]byte(*args.ID))
		if err != nil {
			return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
		}
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE DATAID = ?", dataId)
	} else if args.NftTokenId != nil {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE NFTTOKENID = ?", *args.NftTokenId)
	} else {
		return nil, fmt.Errorf("either ID or nftTokenId must be provided")
	}

	v, err := verseFromRow(row)
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
			if args.UserDataId != nil {
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
			}
			if err != nil {
				return nil, err
			}

			v.IsPaid = count > 0
		}
	}

	return v, nil
}

func (r *resolver) Verses(ctx context.Context, args VerseArgs) ([]*verse, error) {
	// Prepare the base query
	query := "SELECT * FROM VERSE"

	// Add filters if provided
	var filters []string
	if args.Owner != nil {
		filters = append(filters, fmt.Sprintf("OWNER = '%s'", *args.Owner))
	}
	if args.Price != nil {
		filters = append(filters, fmt.Sprintf("PRICE = '%s'", *args.Price))
	}
	if args.CreatedAt != nil {
		filters = append(filters, fmt.Sprintf("CREATEDAT >= %d", *args.CreatedAt))
	}
	if args.Status != nil {
		filters = append(filters, fmt.Sprintf("STATUS = '%s'", *args.Status))
	}

	// Combine the base query with filters
	if len(filters) > 0 {
		query = query + " WHERE " + strings.Join(filters, " AND ")
	}

	query = query + " ORDER BY CREATEDAT DESC"

	rows, err := r.indexSvc.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var verses []*verse
	for rows.Next() {
		v, err := verseFromRow(rows)
		if err != nil {
			return nil, err
		}

		if args.UserDataId != nil {
			count := 0
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
			if err != nil {
				return nil, err
			}
			v.IsPaid = count > 0
		}

		verses = append(verses, v)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

func verseFromRow(rowScanner interface{}) (*verse, error) {
	var v verse

	var err error
	switch scanner := rowScanner.(type) {
	case *sql.Row:
		err = scanner.Scan(
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
	case *sql.Rows:
		err = scanner.Scan(
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
	default:
		return nil, fmt.Errorf("unsupported row scanner type")
	}

	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (v *verse) ID() graphql.ID {
	return graphql.ID(v.CommitId)
}
