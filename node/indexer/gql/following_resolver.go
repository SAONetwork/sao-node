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
	HasFollowed bool       `json:"HasFollowed"`
	Status    string       `json:"Status"`
	ToPay     string         `json:"ToPay"`

	// user profile fields
	EthAddr  string `json:"EthAddr"`
	Avatar   string `json:"Avatar"`
	Username string `json:"Username"`
	Bio      string `json:"Bio"`
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
	row := r.indexSvc.Db.QueryRowContext(ctx, `SELECT UF.*, COALESCE(UP.ETHADDR, ''), COALESCE(UP.AVATAR, ''), COALESCE(UP.USERNAME, ''), COALESCE(UP.BIO, '')
                                                FROM USER_FOLLOWING UF
                                                JOIN USER_PROFILE UP ON UF.FOLLOWER = UP.DATAID
                                                WHERE UF.DATAID = ? AND UF.STATUS = 1`, id)
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
		// user profile fields
		&uf.EthAddr,
		&uf.Avatar,
		&uf.Username,
		&uf.Bio,
	)
	if err != nil {
		return nil, err // return error if query failed
	}

	// return the UserFollowing variable
	return &uf, nil
}

func (r *resolver) Followings(ctx context.Context, args struct {
	FollowingDataId string
	MutualWithId    *string
	Limit           *int32
	Offset          *int32
	UserDataId      *string
}) (*followingResult, error) {
	var rows *sql.Rows
	var err error
	limit := 10 // default limit
	offset := 0 // default offset

	if args.Limit != nil {
		limit = int(*args.Limit)
	}
	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	var countQuery string
	var totalCount int
	if args.MutualWithId != nil {
		countQuery = `SELECT COUNT(*) 
				  FROM USER_FOLLOWING UF1
				  JOIN USER_FOLLOWING UF2 ON UF1.FOLLOWER = UF2.FOLLOWER
				  WHERE UF1.FOLLOWING = ? 
				  AND UF2.FOLLOWING = ?
				  AND UF1.STATUS = 1
				  AND UF2.STATUS = 1`
		err = r.indexSvc.Db.QueryRowContext(ctx, countQuery, args.FollowingDataId, *args.MutualWithId).Scan(&totalCount)
	} else {
		countQuery = `SELECT COUNT(*) 
			FROM USER_FOLLOWING UF
			WHERE UF.FOLLOWING = ? AND UF.STATUS = 1`
		err = r.indexSvc.Db.QueryRowContext(ctx, countQuery, args.FollowingDataId).Scan(&totalCount)

	}

	if err != nil {
		return nil, err
	}

	if args.MutualWithId != nil {
		query := `SELECT UF1.*, COALESCE(UP.ETHADDR, ''), COALESCE(UP.AVATAR, ''), COALESCE(UP.USERNAME, ''), COALESCE(UP.BIO, '') 
			  FROM USER_FOLLOWING UF1
			  JOIN USER_FOLLOWING UF2 ON UF1.FOLLOWER = UF2.FOLLOWER
			  JOIN USER_PROFILE UP ON UF1.FOLLOWER = UP.DATAID
			  WHERE UF1.FOLLOWING = ? 
			  AND UF2.FOLLOWING = ?
			  AND UF1.STATUS = 1
			  AND UF2.STATUS = 1
			  LIMIT ? OFFSET ?`

		rows, err = r.indexSvc.Db.QueryContext(ctx, query, args.FollowingDataId, *args.MutualWithId, limit, offset)

	} else {
		query := `SELECT UF.*, COALESCE(UP.ETHADDR, ''), COALESCE(UP.AVATAR, ''), COALESCE(UP.USERNAME, ''), COALESCE(UP.BIO, '')
		FROM USER_FOLLOWING UF
		JOIN USER_PROFILE UP ON UF.FOLLOWER = UP.DATAID
		WHERE UF.FOLLOWING = ? AND UF.STATUS = 1
		LIMIT ? OFFSET ?`
		rows, err = r.indexSvc.Db.QueryContext(ctx, query, args.FollowingDataId, limit, offset)
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
			// New fields
			&uf.EthAddr,
			&uf.Avatar,
			&uf.Username,
			&uf.Bio,
		)
		if err != nil {
			return nil, err
		}

		// Check if the current user has followed this user
		if args.UserDataId != nil {
			if *args.UserDataId == uf.Follower {
				uf.HasFollowed = true
			} else {
				var followStatus int
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT STATUS FROM USER_FOLLOWING WHERE FOLLOWING = ? AND FOLLOWER = ?", uf.Follower, *args.UserDataId).Scan(&followStatus)
				if err != nil && err != sql.ErrNoRows {
					// If the error is something other than 'no rows', return the error
					fmt.Printf("Error checking follow status: %v\n", err)
				}
				// If a following relationship exists (status = 1), set HasFollowed to true
				uf.HasFollowed = followStatus == 1

				if !uf.HasFollowed {
					sixMonthsAgo := time.Now().AddDate(0, -6, 0).Unix() // 6 months ago in Unix time
					var price sql.NullString
					err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT PRICE FROM LISTING_INFO WHERE ITEMDATAID = ? AND TIME >= ? ORDER BY TIME DESC LIMIT 1", uf.Follower, sixMonthsAgo).Scan(&price)
					if err != nil && err != sql.ErrNoRows {
						// If the error is something other than 'no rows', return the error
						fmt.Printf("Error fetching listing price: %v\n", err)
					}
					// If the price exists, set ToPay to the fetched price
					if price.Valid {
						uf.ToPay = price.String
					}
				}

			}
		}
		followings = append(followings, &uf)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	result := &followingResult{
		Followings: followings,
		Count:      int32(totalCount),
	}

	return result, nil
}

func (r *resolver) FollowedList(ctx context.Context, args struct {
	Follower  string
	IsExpired bool
	Limit     *int32
	Offset    *int32
	UserDataId *string
}) (*followingResult, error) {
	var query, countQuery string
	limit := 10 // default limit
	offset := 0 // default offset

	if args.Limit != nil {
		limit = int(*args.Limit)
	}
	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	if args.IsExpired {
		query = `
		SELECT UF.*, COALESCE(UP.ETHADDR, ''), COALESCE(UP.AVATAR, ''), COALESCE(UP.USERNAME, ''), COALESCE(UP.BIO, '')
		FROM USER_FOLLOWING UF
		LEFT JOIN USER_PROFILE UP ON UF.FOLLOWING = UP.DATAID
		WHERE UF.FOLLOWER = ? AND UF.EXPIREDAT < ? AND UF.EXPIREDAT != 0 AND UF.STATUS = 1
		LIMIT ? OFFSET ?
		`

		countQuery = `
		SELECT COUNT(*) 
		FROM USER_FOLLOWING UF
		WHERE UF.FOLLOWER = ? AND UF.EXPIREDAT < ? AND UF.EXPIREDAT != 0 AND UF.STATUS = 1
		`
	} else {
		query = `
		SELECT UF.*, COALESCE(UP.ETHADDR, ''), COALESCE(UP.AVATAR, ''), COALESCE(UP.USERNAME, ''), COALESCE(UP.BIO, '')
		FROM USER_FOLLOWING UF
		LEFT JOIN USER_PROFILE UP ON UF.FOLLOWING = UP.DATAID
		WHERE UF.FOLLOWER = ? AND (UF.EXPIREDAT > ? OR UF.EXPIREDAT = 0) AND UF.STATUS = 1
		LIMIT ? OFFSET ?
		`

		countQuery = `
		SELECT COUNT(*) 
		FROM USER_FOLLOWING UF
		WHERE UF.FOLLOWER = ? AND (UF.EXPIREDAT > ? OR UF.EXPIREDAT = 0) AND UF.STATUS = 1
		`
	}

	currentTime := time.Now().Unix()
	var totalCount int
	err := r.indexSvc.Db.QueryRowContext(ctx, countQuery, args.Follower, currentTime).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, query, args.Follower, currentTime, limit, offset)
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
			// user profile fields
			&uf.EthAddr,
			&uf.Avatar,
			&uf.Username,
			&uf.Bio,
		)
		if err != nil {
			return nil, err
		}

		// Check if the current user has followed this user
		if args.UserDataId != nil {
			if *args.UserDataId == uf.Following {
				uf.HasFollowed = true
			} else {
				var followStatus int
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT STATUS FROM USER_FOLLOWING WHERE FOLLOWING = ? AND FOLLOWER = ?", uf.Following, *args.UserDataId).Scan(&followStatus)
				if err != nil && err != sql.ErrNoRows {
					fmt.Printf("Error checking follow status: %v\n", err)
				}
				// If a following relationship exists (status = 1), set HasFollowed to true
				uf.HasFollowed = followStatus == 1

				// Check if the current user needs to pay to follow this user
				if !uf.HasFollowed {
					sixMonthsAgo := time.Now().AddDate(0, -6, 0).Unix() // 6 months ago in Unix time
					var price sql.NullString
					err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT PRICE FROM LISTING_INFO WHERE ITEMDATAID = ? AND TIME >= ? ORDER BY TIME DESC LIMIT 1", uf.Following, sixMonthsAgo).Scan(&price)
					if err != nil && err != sql.ErrNoRows {
						// If the error is something other than 'no rows', return the error
						fmt.Printf("Error fetching listing price: %v\n", err)
					}
					// If the price exists, set ToPay to the fetched price
					if price.Valid {
						uf.ToPay = price.String
					}
				}
			}
		}
		followedList = append(followedList, &uf)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	result := &followingResult{
		Followings: followedList,
		Count:      int32(totalCount),
	}

	return result, nil
}

func (m *userFollowing) ID() graphql.ID {
	return graphql.ID(m.CommitId)
}
