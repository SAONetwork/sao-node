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
	Size          string
	Access		  string
}

type metadataList struct {
	TotalCount int32
	Metadatas  []*metadata
	More       bool
}

type Group struct {
	GroupId    string `json:"groupId"`
	LastChange int32  `json:"lastChange"`
	Files      int32  `json:"files"`
}

type GroupList struct {
	Groups []*Group
}

type UserSummary struct {
	GroupCount int32
	TotalFiles int32
	Expiration int32
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

	sizeRow := r.indexSvc.Db.QueryRowContext(ctx, `SELECT size FROM ORDERS WHERE id= ?`, meta.OrderId)
	err = sizeRow.Scan(&meta.Size)
	if err != nil {
		log.Errorf("database scan error - size: %v", err)
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
			log.Errorf("database scan error: %v", err)
			continue
		}

		sizeRow := r.indexSvc.Db.QueryRowContext(ctx, `SELECT size FROM ORDERS WHERE id= ?`, meta.OrderId)
		err = sizeRow.Scan(&meta.Size)
		if err != nil {
			log.Errorf("database scan error - size: %v", err)
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

// GetGroupList fetches the group list by groupId.
func (r *resolver) GetGroupList(ctx context.Context) (*GroupList, error) {
	rows, err := r.indexSvc.Db.QueryContext(ctx, `
		SELECT groupId, MAX(createdAt) as createdAt, COUNT(*) as files 
		FROM METADATA
		GROUP BY groupId`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]*Group, 0)
	for rows.Next() {
		group := &Group{}
		if err := rows.Scan(&group.GroupId, &group.LastChange, &group.Files); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &GroupList{
		Groups: groups,
	}, nil
}

// GetUserSummary fetches the user summary for a specific owner.
func (r *resolver) GetUserSummary(ctx context.Context, owner string) (*UserSummary, error) {
	lastHeight, err := r.chainSvc.GetLastHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get last height: %w", err)
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, `
        SELECT COUNT(DISTINCT groupId) as GroupCount, COUNT(*) as TotalFiles,
            SUM(CASE WHEN duration < ? THEN 1 ELSE 0 END) as Expiration
        FROM METADATA
        WHERE owner = ?`, lastHeight, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := &UserSummary{}
	if rows.Next() {
		if err := rows.Scan(&summary.GroupCount, &summary.TotalFiles, &summary.Expiration); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no summary available for the given owner")
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return summary, nil
}


func (r *resolver) MetadataCount(ctx context.Context) (int32, error) {
	return 0, nil
}

func (m *metadata) ID() graphql.ID {
	return m.DataId
}
