package gql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
	"strings"
)

type verse struct {
	CommitId        string `json:"CommitId"`
	DataId string `json:"DataId"`
	CreatedAt   types.Uint64
	FileIDs     []string
	Owner       string
	Price       string
	Digest      string
	Scope       string
	Status      string
	NftTokenID  string
}

// query: verse(id) Verse
func (r *resolver) Verse(ctx context.Context, args struct{ ID graphql.ID }) (*verse, error) {
	var commitId uuid.UUID
	err := commitId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE COMMITID = ?", commitId)
	return verseFromRow(row)
}

func (r *resolver) Verses(ctx context.Context, args struct {
	Owner     *string
	Price     *string
	CreatedAt *types.Uint64
	Status    *string
}) ([]*verse, error) {
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
		verses = append(verses, v)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

func verseFromRow(rowScanner interface{}) (*verse, error) {
	var v verse
	var fileIDsJSON string

	var err error
	switch scanner := rowScanner.(type) {
	case *sql.Row:
		err = scanner.Scan(
			&v.CommitId,
			&v.DataId,
			&v.CreatedAt,
			&fileIDsJSON,
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
			&v.CreatedAt,
			&fileIDsJSON,
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