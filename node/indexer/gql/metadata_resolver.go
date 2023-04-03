package gql

import (
	"context"

	"github.com/graph-gophers/graphql-go"
)

type metadata struct {
	CommitId   string
	Did        string
	DataId     string
	Alias      string
	GroupId    string
	Version    string
	Size       uint
	Expiration uint64
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
	return &metadata{
		CommitId: "",
	}, nil
}

type metadatasArgs struct {
	Query  graphql.NullString
	Cursor *graphql.ID
	Offset graphql.NullInt
	Limit  graphql.NullInt
}

// query: metadatas(cursor, offset, limit) DealList
func (r *resolver) Metadatas(ctx context.Context, args metadatasArgs) (*metadataList, error) {
	metadatas := make([]*metadata, 0)

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
