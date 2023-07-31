package gql

import (
	"context"
	"fmt"

	"github.com/graph-gophers/graphql-go"
)

type metadata struct {
	DataId        graphql.ID
	Owner         string
	Alias         string
	GroupId       string
	OrderId       string
	Tags          string
	Cid           string
	Commits       string
	ExtendInfo    string
	Update        bool
	Commit        string
	Rule          string
	Duration      int32
	CreatedAt     int32
	ReadonlyDids  string
	ReadwriteDids string
	Status        int32
	Orders        string
}

type metadataList struct {
	TotalCount int32
	Metadatas  []*metadata
	More       bool
}

type QueryArgs struct {
	Query  graphql.NullString
	Owner  graphql.NullString
}

// query: metadata(dataId) Metadata
func (r *resolver) Metadata(ctx context.Context, args struct{ ID graphql.ID }) (*metadata, error) {
	var dataId string
	dataId = string(args.ID)

	row := r.indexSvc.Db.QueryRowContext(ctx, `SELECT dataId, owner, alias, groupId, orderId, tags, cid, commits, extendInfo, 
    updateAt, commitId, rule, duration, createdAt, readonlyDids, readwriteDids, status, orders 
    FROM METADATA WHERE dataId= ?`, dataId)

	meta := &metadata{}
	err := row.Scan(&meta.DataId, &meta.Owner, &meta.Alias, &meta.GroupId, &meta.OrderId, &meta.Tags, &meta.Cid,
		&meta.Commits, &meta.ExtendInfo, &meta.Update, &meta.Commit, &meta.Rule, &meta.Duration, &meta.CreatedAt,
		&meta.ReadonlyDids, &meta.ReadwriteDids, &meta.Status, &meta.Orders)
	if err != nil {
		return nil, fmt.Errorf("database scan error: %v", err)
	}

	return meta, nil
}

// query: metadatas(Query) MetaList
func (r *resolver) Metadatas(ctx context.Context, args QueryArgs) (*metadataList, error) {
	queryStr := `SELECT dataId, owner, alias, groupId, orderId, tags, cid, commits, extendInfo, 
    updateAt, commitId, rule, duration, createdAt, readonlyDids, readwriteDids, status, orders FROM METADATA`

	var params []interface{}

	if args.Query.Set && args.Query.Value != nil {
		queryStr += " WHERE " + *args.Query.Value
	}

	if args.Owner.Set && args.Owner.Value != nil {
		if len(params) > 0 {
			queryStr += " AND "
		} else {
			queryStr += " WHERE "
		}
		queryStr += "owner = ?"
		params = append(params, *args.Owner.Value)
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, queryStr, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadatas := make([]*metadata, 0)
	for rows.Next() {
		meta := &metadata{}
		err = rows.Scan(&meta.DataId, &meta.Owner, &meta.Alias, &meta.GroupId, &meta.OrderId, &meta.Tags, &meta.Cid,
			&meta.Commits, &meta.ExtendInfo, &meta.Update, &meta.Commit, &meta.Rule, &meta.Duration, &meta.CreatedAt,
			&meta.ReadonlyDids, &meta.ReadwriteDids, &meta.Status, &meta.Orders)
		if err != nil {
			return nil, err
		}
		metadatas = append(metadatas, meta)
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
	return m.DataId
}
