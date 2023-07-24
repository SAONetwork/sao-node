package gql

import (
	"context"
	"fmt"

	"github.com/SaoNetwork/sao-node/node/indexer/gql/types"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
)

type metadata struct {
	CommitId   string
	Did        string
	DataId     string
	Alias      string
	Cid        string
	GroupId    string
	Version    string
	Size       int32
	Expiration types.Uint64
	Readers    string
	Writers    string
}

type metadataList struct {
	TotalCount int32
	Metadatas  []*metadata
	More       bool
}

// query: metadata(id) Metadata
func (r *resolver) Metadata(ctx context.Context, args struct{ ID graphql.ID }) (*metadata, error) {
	var commitId uuid.UUID
	err := commitId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT COMMITID, DID, COALESCE(CID, '') AS CID, DATAID, ALIAS, PLAT, VER, SIZE, EXPIRATION, READER, WRITER FROM METADATA WHERE COMMITID= ?", commitId.String())
	var CommitId string
	var Did string
	var DataId string
	var Cid string
	var Alias string
	var GroupId string
	var Version string
	var Size int32
	var Expiration types.Uint64
	var Readers string
	var Writers string
	err = row.Scan(&CommitId, &Did, &Cid, &DataId, &Alias, &GroupId, &Version, &Size, &Expiration, &Readers, &Writers)
	if err != nil {
		fmt.Errorf("database scan error: %v", err)
		return nil, err
	}

	return &metadata{
		CommitId, Did, Cid, DataId, Alias, GroupId, Version, Size, Expiration, Readers, Writers,
	}, nil
}

// query: metadatas(cursor, offset, limit) MetaList
func (r *resolver) Metadatas(ctx context.Context, args struct{ Query graphql.NullString }) (*metadataList, error) {
	queryStr := "SELECT COMMITID, DID, CID, DATAID, ALIAS, PLAT, VER, SIZE, EXPIRATION, READER, WRITER FROM METADATA "
	if args.Query.Set && args.Query.Value != nil {
		queryStr = queryStr + *args.Query.Value
	}
	rows, err := r.indexSvc.Db.QueryContext(ctx, queryStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadatas := make([]*metadata, 0)
	for rows.Next() {
		var CommitId string
		var Did string
		var Cid string
		var DataId string
		var Alias string
		var GroupId string
		var Version string
		var Size int32
		var Expiration types.Uint64
		var Readers string
		var Writers string
		err = rows.Scan(&CommitId, &Did, &Cid, &DataId, &Alias, &GroupId, &Version, &Size, &Expiration, &Readers, &Writers)
		if err != nil {
			return nil, err
		}
		metadatas = append(metadatas, &metadata{
			CommitId, Did, Cid, DataId, Alias, GroupId, Version, Size, Expiration, Readers, Writers,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &metadataList{
		TotalCount: int32(len(metadatas)),
		Metadatas:  metadatas,
		More:       false,
	}, nil
}

func (r *resolver) MetadataCount(ctx context.Context) (int32, error) {
	return 0, nil
}

func (m *metadata) ID() graphql.ID {
	return graphql.ID(m.CommitId)
}
