package gql

import (
	"context"
	"fmt"
	"strings"

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
	Access        string
}

type metadataList struct {
	TotalCount int32
	Metadatas  []*metadata
	More       bool
}

type CommitInfo struct {
	CommitId  string
	Size      string
	CreatedAt int32
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
	GroupCount   int32
	TotalFiles   int32
	Expiration   int32
	TotalStorage string
	TotalSpent   string
	Balance      string
}

type QueryArgs struct {
	Query   graphql.NullString
	Owner   graphql.NullString
	GroupId graphql.NullString
	Limit   *int32
	Offset  *int32
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
	baseQuery := `FROM METADATA`
	queryStr := `SELECT dataId, owner, alias, groupId, orderId, tags, cid, commits, extendInfo, 
    updateAt, commitId, rule, duration, createdAt, readonlyDids, readwriteDids, status, orders ` + baseQuery

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

	if args.GroupId.Set && args.GroupId.Value != nil {
		if len(params) > 0 {
			queryStr += " AND "
		} else {
			queryStr += " WHERE "
		}
		queryStr += "groupId = ?"
		params = append(params, args.GroupId.Value)
	}

	queryStr += " ORDER BY createdAt DESC"

	// Add limit and offset if provided
	if args.Limit != nil {
		queryStr += " LIMIT ?"
		params = append(params, int(*args.Limit))
	}
	if args.Offset != nil {
		queryStr += " OFFSET ?"
		params = append(params, int(*args.Offset))
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

		if strings.Contains(meta.ReadonlyDids, "did:key:zQ3shggYEtCZNEiwSeqLdLo97SqS2ERMHB2mgV8hmCGDn4DJ3") {
			meta.Access = "public" // Set Access to "public"
		} else {
			meta.Access = "private"
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch total count for pagination
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int32
	err = r.indexSvc.Db.QueryRowContext(ctx, countQuery, params...).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	moreResults := false
	if args.Limit != nil && len(metadatas) == int(*args.Limit) {
		moreResults = true
	}

	return &metadataList{
		TotalCount: totalCount,
		Metadatas:  metadatas,
		More:       moreResults, // if the number of fetched rows is equal to the limit, then there might be more results.
	}, nil
}

func (r *resolver) Commits(ctx context.Context, args struct{ DataId string }) ([]CommitInfo, error) {
	// Get the commits field for the given dataId
	query := `SELECT commits FROM METADATA WHERE dataId = ?`
	row := r.indexSvc.Db.QueryRowContext(ctx, query, args.DataId)

	var commitIdString string
	if err := row.Scan(&commitIdString); err != nil {
		return nil, err
	}

	// Split the commits field by comma
	commitPairs := strings.Split(commitIdString, ",")

	var commits []CommitInfo
	for _, pair := range commitPairs {
		// Split each pair by the special character to get the commitId and height
		parts := strings.Split(pair, "")
		if len(parts) < 2 {
			fmt.Sprintf("commitId and height not formatted in %s", pair)
		}
		commitId := parts[0]

		query = `SELECT size, createdAt FROM ORDERS WHERE commitId = ?`
		row = r.indexSvc.Db.QueryRowContext(ctx, query, commitId)

		var commit CommitInfo
		if err := row.Scan(&commit.Size, &commit.CreatedAt); err != nil {
			fmt.Errorf("database scan error: %v", err)
			continue
		}
		commit.CommitId = commitId

		commits = append(commits, commit)
	}

	return commits, nil
}

// GroupList fetches the group list by groupId.
func (r *resolver) GroupList(ctx context.Context, args struct{ Owner string }) (*GroupList, error) {
	rows, err := r.indexSvc.Db.QueryContext(ctx, `
		SELECT groupId, MAX(createdAt) as createdAt, COUNT(*) as files 
		FROM METADATA
		WHERE owner = ?
		GROUP BY groupId`, args.Owner)
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

// UserSummary fetches the user summary for a specific owner.
func (r *resolver) UserSummary(ctx context.Context, args struct {
	Owner   string
	Address *string
}) (*UserSummary, error) {
	lastHeight, err := r.chainSvc.GetLastHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get last height: %w", err)
	}

	// Query METADATA table
	metaRow, err := r.indexSvc.Db.QueryContext(ctx, `
		SELECT COUNT(DISTINCT groupId) as GroupCount, COUNT(*) as TotalFiles,
			SUM(CASE WHEN duration < ? THEN 1 ELSE 0 END) as Expiration
		FROM METADATA
		WHERE owner = ?`, lastHeight, args.Owner)
	if err != nil {
		return nil, err
	}
	defer metaRow.Close()

	summary := &UserSummary{}
	if metaRow.Next() {
		if err := metaRow.Scan(&summary.GroupCount, &summary.TotalFiles, &summary.Expiration); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no metadata summary available for the given owner")
	}

	if err := metaRow.Err(); err != nil {
		return nil, err
	}

	// Query ORDERS table
	orderRow, err := r.indexSvc.Db.QueryContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN status = 3 THEN size ELSE 0 END), 0) as TotalStorage,
			COALESCE(SUM(CASE WHEN status = 3 THEN amount ELSE 0 END), 0) as TotalSpent
		FROM ORDERS
		WHERE owner = ?`, args.Owner)
	if err != nil {
		return nil, err
	}
	defer orderRow.Close()

	if orderRow.Next() {
		if err := orderRow.Scan(&summary.TotalStorage, &summary.TotalSpent); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no order summary available for the given owner")
	}

	if err := orderRow.Err(); err != nil {
		return nil, err
	}

	if args.Address != nil {
		balance, err := r.chainSvc.GetBalance(ctx, *args.Address)
		if err != nil {
			return nil, fmt.Errorf("unable to get balance: %w", err)
		}
		summary.Balance = balance[0].Amount.String()
	}

	return summary, nil
}

func (r *resolver) MetadataCount(ctx context.Context) (int32, error) {
	return 0, nil
}

func (m *metadata) ID() graphql.ID {
	return m.DataId
}
