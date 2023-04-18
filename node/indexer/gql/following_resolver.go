package gql

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type userFollowing struct {
	CommitId        string `json:"CommitId"`
	DataId          string `json:"DataId"`
	CreatedAt types.Uint64  `json:"CreatedAt"`
	UpdatedAt types.Uint64  `json:"UpdatedAt"`
	ExpiredAt types.Uint64  `json:"ExpiredAt"`
	Follower  string `json:"Follower"`
	Following string `json:"Following"`
	Status    string    `json:"Status"`
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
	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM USER_FOLLOWING WHERE COMMITID = ?", id)
	err = row.Scan(
		&uf.CommitId,
		&uf.DataId,
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

func (r *resolver) Followings(ctx context.Context, args struct{ FollowingDataId string }) (*followingResult, error) {
	rows, err := r.indexSvc.Db.QueryContext(ctx, "SELECT * FROM USER_FOLLOWING WHERE FOLLOWING = ?", args.FollowingDataId)
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
		Count: int32(count),
	}

	return result, nil
}

func (m *userFollowing) ID() graphql.ID {
	return graphql.ID(m.CommitId)
}
