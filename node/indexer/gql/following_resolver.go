package gql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
	"time"
)

type userFollowing struct {
	CommitId  string       `json:"CommitId"`
	DataId    string       `json:"DataId"`
	Alias     string       `json:"Alias"`
	CreatedAt types.Uint64 `json:"CreatedAt"`
	UpdatedAt types.Uint64 `json:"UpdatedAt"`
	ExpiredAt types.Uint64 `json:"ExpiredAt"`
	Follower  string       `json:"Follower"`
	Following string       `json:"Following"`
	Status    string       `json:"Status"`
	ToPay     bool         `json:"ToPay"`
}

type followingResult struct {
	Followings []*userFollowing
	Count      int32
}

// query: userFollowing(id) UserFollowing
func (r *resolver) UserFollowing(ctx context.Context, args struct{ ID graphql.ID }) (*userFollowing, error) {
	var id uuid.UUID
	err := id.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	// query the database for the user following with the given id
	var uf userFollowing
	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM USER_FOLLOWING WHERE DATAID = ? AND STATUS = 1", id)
	err = row.Scan(
		&uf.CommitId,
		&uf.DataId,
		&uf.Alias,
		&uf.CreatedAt,
		&uf.UpdatedAt,
		&uf.ExpiredAt,
		&uf.Follower,
		&uf.Following,
		&uf.Status,
	)
	if err != nil {
		return nil, err // return error if query failed
	}

	// return the UserFollowing variable
	return &uf, nil
}

func (r *resolver) Followings(ctx context.Context, args struct{ FollowingDataId string
	MutualWithId *string }) (*followingResult, error) {

	var rows *sql.Rows
	var err error

	if args.MutualWithId != nil {
		query := `SELECT UF1.* 
			  FROM USER_FOLLOWING UF1
			  JOIN USER_FOLLOWING UF2
			  ON UF1.FOLLOWER = UF2.FOLLOWER
			  WHERE UF1.FOLLOWING = ? 
			  AND UF2.FOLLOWING = ?
			  AND UF1.STATUS = 1
			  AND UF2.STATUS = 1`

		rows, err = r.indexSvc.Db.QueryContext(ctx, query, args.FollowingDataId, *args.MutualWithId)

	} else {
		query := `SELECT * FROM USER_FOLLOWING WHERE FOLLOWING = ? AND STATUS = 1`
		rows, err = r.indexSvc.Db.QueryContext(ctx, query, args.FollowingDataId)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followings []*userFollowing
	for rows.Next() {
		var uf userFollowing
		err = rows.Scan(
			&uf.CommitId,
			&uf.DataId,
			&uf.Alias,
			&uf.CreatedAt,
			&uf.UpdatedAt,
			&uf.ExpiredAt,
			&uf.Follower,
			&uf.Following,
			&uf.Status,
		)
		if err != nil {
			return nil, err
		}
		followings = append(followings, &uf)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	count := len(followings)
	result := &followingResult{
		Followings: followings,
		Count:      int32(count),
	}

	return result, nil
}

func (r *resolver) FollowedList(ctx context.Context, args struct{ Follower string; IsExpired bool }) (*followingResult, error) {
	var query string
	if args.IsExpired {
		query = "SELECT * FROM USER_FOLLOWING WHERE FOLLOWER = ? AND EXPIREDAT < ? AND EXPIREDAT != 0 AND STATUS = 1"
	} else {
		query = "SELECT * FROM USER_FOLLOWING WHERE FOLLOWER = ? AND (EXPIREDAT > ? OR EXPIREDAT = 0) AND STATUS = 1"
	}

	currentTime := time.Now().Unix()
	rows, err := r.indexSvc.Db.QueryContext(ctx, query, args.Follower, currentTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followedList []*userFollowing
	for rows.Next() {
		var uf userFollowing
		err = rows.Scan(
			&uf.CommitId,
			&uf.DataId,
			&uf.Alias,
			&uf.CreatedAt,
			&uf.UpdatedAt,
			&uf.ExpiredAt,
			&uf.Follower,
			&uf.Following,
			&uf.Status,
		)
		if err != nil {
			return nil, err
		}
		followedList = append(followedList, &uf)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	count := len(followedList)
	result := &followingResult{
		Followings: followedList,
		Count:      int32(count),
	}

	return result, nil
}


func (m *userFollowing) ID() graphql.ID {
	return graphql.ID(m.CommitId)
}
