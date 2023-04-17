package gql

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type fileInfo struct {
	CommitId     string
	DataID       string
	CreatedAt    types.Uint64
	FileDataID   string
	ContentType  string
	Owner        string
	Filename     string
	FileCategory string
}

// query: fileInfo(id) FileInfo
func (r *resolver) FileInfo(ctx context.Context, args struct{ ID graphql.ID }) (*fileInfo, error) {
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

	return &fi, nil
}

func (fi *fileInfo) ID() graphql.ID {
	return graphql.ID(fi.CommitId)
}
