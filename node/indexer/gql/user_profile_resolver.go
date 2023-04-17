package gql

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type userProfile struct {
	CommitId        string `json:"CommitId"`
	DataId          string `json:"DataId"`
	CreatedAt       types.Uint64  `json:"CreatedAt"`
	UpdatedAt       types.Uint64  `json:"UpdatedAt"`
	Did             string `json:"Did"`
	EthAddr         string `json:"EthAddr"`
	Avatar          string `json:"Avatar"`
	Username        string `json:"Username"`
	FollowingCount  int32 `json:"FollowingCount"`
	Twitter         string `json:"Twitter"`
	Youtube         string `json:"Youtube"`
	Bio             string `json:"Bio"`
	Banner          string `json:"Banner"`
	FollowingDataId string `json:"FollowingDataId"`
}

// query: userProfile(id) UserProfile
func (r *resolver) UserProfile(ctx context.Context, args struct{ ID graphql.ID }) (*userProfile, error) {
	var commitId uuid.UUID
	err := commitId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	// query the database for the user profile with the given commitId
	var profile userProfile
	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM USER_PROFILE WHERE COMMITID = ?", commitId) // use r.indexSvc.Db.QueryRowContext instead of db.QueryRow
	err = row.Scan(
		&profile.CommitId,
		&profile.DataId,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&profile.Did,
		&profile.EthAddr,
		&profile.Avatar,
		&profile.Username,
		&profile.FollowingCount,
		&profile.Twitter,
		&profile.Youtube,
		&profile.Bio,
		&profile.Banner,
		&profile.FollowingDataId,
	)
	if err != nil {
		return nil, err // return error if query failed
	}

	// return the UserProfile variable
	return &profile, nil
}

func (m *userProfile) ID() graphql.ID {
	return graphql.ID(m.CommitId)
}
