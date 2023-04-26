package gql

import (
	"context"
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
	ID        *graphql.ID
	Did       *string
	EthAddress *string
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

	query := "SELECT * FROM USER_PROFILE WHERE"
	queryParams := make([]interface{}, 0)
	argsCount := 0

	if args.ID != nil {
		query += " DATAID = ?"
		queryParams = append(queryParams, dataId)
		argsCount++
	}

	if args.Did != nil {
		if argsCount > 0 {
			query += " OR"
		}
		query += " DID = ?"
		queryParams = append(queryParams, *args.Did)
		argsCount++
	}

	if args.EthAddress != nil {
		if argsCount > 0 {
			query += " OR"
		}
		query += " ETHADDR = ?"
		queryParams = append(queryParams, *args.EthAddress)
		argsCount++
	}

	if argsCount == 0 {
		return nil, fmt.Errorf("either ID, DID, or EthAddress must be provided")
	}

	row := r.indexSvc.Db.QueryRowContext(ctx, query, queryParams...)

	var profile userProfile
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
		return nil, err
	}

	return &profile, nil
}

func (m *userProfile) ID() graphql.ID {
	return graphql.ID(m.CommitId)
}
