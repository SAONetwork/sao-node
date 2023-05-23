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
	FollowerCount   int32        `json:"FollowerCount"`
	Twitter         string       `json:"Twitter"`
	Youtube         string       `json:"Youtube"`
	Bio             string       `json:"Bio"`
	Banner          string       `json:"Banner"`
	FollowingDataId string       `json:"FollowingDataId"`
	IsFollowing     bool         `json:"IsFollowing"`
	NftTokenID      string       `json:"NftTokenId"`
	PayToFollow     string       `json:"PayToFollow"`
}

type userProfileArgs struct {
	ID         *graphql.ID
	Did        *string
	EthAddress *string
}

type suggestedUsersArgs struct {
	UserDataId *string
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

	// Get the FollowingCount
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWER = ?", profile.DataId).Scan(&profile.FollowingCount)
	if err != nil {
		return nil, err
	}

	// Get the FollowerCount
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWING = ?", profile.DataId).Scan(&profile.FollowerCount)
	if err != nil {
		return nil, err
	}

	// Fetch the NFT token ID and price
	var tokenID, price sql.NullString
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT TOKENID, PRICE FROM LISTING_INFO WHERE ITEMDATAID = ? ORDER BY TIME DESC LIMIT 1", profile.DataId).Scan(&tokenID, &price)
	if err != nil && err != sql.ErrNoRows {
		// If the error is something other than 'no rows', return the error
		fmt.Printf("Error fetching token ID and price: %v\n", err)
	} else {
		if tokenID.Valid {
			profile.NftTokenID = tokenID.String
		}
		// If the price exists, set PayToFollow to the fetched price
		if price.Valid {
			profile.PayToFollow = price.String
		}
	}

	return &profile, nil
}

// query: suggestedUsers(userDataId) [UserProfile!]!
func (r *resolver) SuggestedUsers(ctx context.Context, args suggestedUsersArgs) ([]*userProfile, error) {
	query := `SELECT USER_PROFILE.* 
          FROM USER_PROFILE 
          LEFT JOIN (
            SELECT FOLLOWING, COUNT(*) as COUNT 
            FROM USER_FOLLOWING 
            GROUP BY FOLLOWING
          ) as FOLLOWING_COUNTS ON USER_PROFILE.DATAID = FOLLOWING_COUNTS.FOLLOWING
          WHERE USER_PROFILE.DATAID != ? AND USER_PROFILE.DATAID NOT IN (SELECT ITEMDATAID FROM LISTING_INFO)
          ORDER BY FOLLOWING_COUNTS.COUNT DESC 
          LIMIT 5`

	rows, err := r.indexSvc.Db.QueryContext(ctx, query, *args.UserDataId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestedProfiles []*userProfile
	for rows.Next() {
		var profile userProfile
		err = rows.Scan(
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

		// New query to check if the user specified by args.UserDataId is following the suggested user
		isFollowingQuery := `SELECT COUNT(*) 
							 FROM USER_FOLLOWING 
							 WHERE FOLLOWER = ? AND FOLLOWING = ?`
		var count int
		err = r.indexSvc.Db.QueryRowContext(ctx, isFollowingQuery, args.UserDataId, profile.DataId).Scan(&count)
		if err != nil {
			return nil, err
		}

		// If count is greater than 0, it means the user is following the suggested user
		profile.IsFollowing = count > 0

		// Get the FollowingCount
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWER = ?", profile.DataId).Scan(&profile.FollowingCount)
		if err != nil {
			return nil, err
		}

		// Get the FollowerCount
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWING = ?", profile.DataId).Scan(&profile.FollowerCount)
		if err != nil {
			return nil, err
		}

		// Fetch the NFT token ID and price
		var tokenID, price sql.NullString
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT TOKENID, PRICE FROM LISTING_INFO WHERE ITEMDATAID = ? ORDER BY TIME DESC LIMIT 1", profile.DataId).Scan(&tokenID, &price)
		if err != nil && err != sql.ErrNoRows {
			// If the error is something other than 'no rows', return the error
			fmt.Printf("Error fetching token ID and price: %v\n", err)
		} else {
			if tokenID.Valid {
				profile.NftTokenID = tokenID.String
			}
			// If the price exists, set PayToFollow to the fetched price
			if price.Valid {
				profile.PayToFollow = price.String
			}
		}

		suggestedProfiles = append(suggestedProfiles, &profile)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return suggestedProfiles, nil
}

func (m *userProfile) ID() graphql.ID {
	return graphql.ID(m.CommitId)
}
