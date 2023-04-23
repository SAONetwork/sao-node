package gql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type userProfile struct {
	CommitId        string       `json:"CommitId"`
	DataId          string       `json:"DataId"`
	Alias           string       `json:"Alias"`
	CreatedAt       types.Uint64 `json:"CreatedAt"`
	UpdatedAt       types.Uint64 `json:"UpdatedAt"`
	Did             string       `json:"Did"`
	EthAddr         string       `json:"EthAddr"`
	Avatar          string       `json:"Avatar"`
	Username        string       `json:"Username"`
	FollowingCount  int32        `json:"FollowingCount"`
	Twitter         string       `json:"Twitter"`
	Youtube         string       `json:"Youtube"`
	Bio             string       `json:"Bio"`
	Banner          string       `json:"Banner"`
	FollowingDataId string       `json:"FollowingDataId"`
}

type userProfileArgs struct {
	ID  *graphql.ID
	Did *string
}

// query: userProfile(id) UserProfile
func (r *resolver) UserProfile(ctx context.Context, args userProfileArgs) (*userProfile, error) {
	var dataId uuid.UUID

	if args.ID != nil {
		err := dataId.UnmarshalText([]byte(*args.ID))
		if err != nil {
			return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", *args.ID, err)
		}
	}

	// query the database for the user profile with the given dataId or did
	var profile userProfile
	var row *sql.Row

	if args.ID != nil && args.Did != nil {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM USER_PROFILE WHERE DATAID = ? OR DID = ?", dataId, *args.Did)
	} else if args.ID != nil {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM USER_PROFILE WHERE DATAID = ?", dataId)
	} else if args.Did != nil {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM USER_PROFILE WHERE DID = ?", *args.Did)
	} else {
		return nil, fmt.Errorf("either ID or DID must be provided")
	}

	err := row.Scan(
		&profile.CommitId,
		&profile.DataId,
		&profile.Alias,
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
