package gql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"github.com/SaoNetwork/sao-node/node/indexer/gql/types"
	"strconv"
	"strings"
	"time"
)

type verse struct {
	CommitId         string `json:"CommitId"`
	DataId           string `json:"DataId"`
	Alias            string `json:"Alias"`
	CreatedAt        types.Uint64
	FileIDs          string
	Owner            string
	Price            string
	Digest           string
	Scope            int32
	Status           string
	NftTokenID       string
	FileType         string
	IsPaid           bool
	PaidTime         types.Uint64
	NotInScope       int32
	PayToFollow      string
	CommentCount     int32
	LikeCount        int32
	HasFollowedOwner bool
	HasLiked         bool
	OwnerEthAddr     string
	OwnerAvatar      string
	OwnerUsername    string
	OwnerBio         string
	OwnerNftTokenID  string
}

type VerseArgs struct {
	ID         *graphql.ID
	UserDataId *string
	Owner      *string
	Price      *string
	CreatedAt  *types.Uint64
	Status     *string
	NftTokenId *string
	Limit      *int32
	Offset     *int32
}

type subscribedVersesArgs struct {
	UserDataId *string
	Limit      *int32
	Offset     *int32
}

type likedVersesArgs struct {
	UserDataId *string
	Limit      *int32
	Offset     *int32
}

// query: verse(id) Verse
func (r *resolver) Verse(ctx context.Context, args VerseArgs) (*verse, error) {
	var row *sql.Row
	if args.ID != nil {
		var dataId uuid.UUID
		err := dataId.UnmarshalText([]byte(*args.ID))
		if err != nil {
			return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
		}
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE DATAID = ? AND STATUS !=2 ", dataId)
	} else if args.NftTokenId != nil {
		row = r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE NFTTOKENID = ? AND STATUS !=2 ", *args.NftTokenId)
	} else {
		return nil, fmt.Errorf("either ID or nftTokenId must be provided")
	}

	v, err := verseFromRow(row, ctx, r.indexSvc.Db, args.UserDataId)
	if err != nil {
		return nil, err
	}

	// If verse price is greater than 0, check if there's a PurchaseOrder record with ItemDataID = verse.DATAID and BuyerDataID = userDataId
	if v.Price != "" {
		price, err := strconv.ParseFloat(v.Price, 64)
		if err != nil {
			return nil, err
		}

		if price > 0 {
			var count int
			var paidTime sql.NullInt64
			if args.UserDataId != nil {
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*), MAX(TIME) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count, &paidTime)
			}
			if err != nil {
				return nil, err
			}

			v.IsPaid = count > 0
			if paidTime.Valid {
				v.PaidTime = types.Uint64(paidTime.Int64)
			}
		}
	}

	if args.UserDataId != nil {
		// Process verse scope
		v, err = processVerseScope(ctx, r.indexSvc.Db, v, *args.UserDataId)
		if err != nil {
			return nil, err
		}
	} else {
		if v.Scope == 2 || v.Scope == 3 || v.Scope == 4 {
			v.NotInScope = v.Scope
		}
		if v.Scope == 5 {
			return nil, fmt.Errorf("verse is private")
		}
	}

	return v, nil
}


func (r *resolver) Verses(ctx context.Context, args VerseArgs) ([]*verse, error) {
	// Prepare the base query
	query := "SELECT * FROM VERSE"

	// Add filters if provided
	var filters []string
	if args.Owner != nil {
		filters = append(filters, fmt.Sprintf("OWNER = '%s'", *args.Owner))
	}
	if args.Price != nil {
		filters = append(filters, fmt.Sprintf("PRICE = '%s'", *args.Price))
	}
	if args.CreatedAt != nil {
		filters = append(filters, fmt.Sprintf("CREATEDAT >= %d", *args.CreatedAt))
	}
	if args.Status != nil {
		filters = append(filters, fmt.Sprintf("STATUS = '%s'", *args.Status))
	}

	// Combine the base query with filters
	if len(filters) > 0 {
		query = query + " WHERE " + strings.Join(filters, " AND ")
	}

	query = query + " AND STATUS !=2 ORDER BY CREATEDAT DESC"

	// Add limit and offset if provided
	if args.Limit != nil {
		query = query + fmt.Sprintf(" LIMIT %d", *args.Limit)
	}
	if args.Offset != nil {
		query = query + fmt.Sprintf(" OFFSET %d", *args.Offset)
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var verses []*verse
	for rows.Next() {
		v, err := verseFromRow(rows, ctx, r.indexSvc.Db, args.UserDataId)
		if err != nil {
			return nil, err
		}

		if args.UserDataId != nil {
			count := 0
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
			if err != nil {
				return nil, err
			}
			v.IsPaid = count > 0

			// Process verse scope
			v, err = processVerseScope(ctx, r.indexSvc.Db, v, *args.UserDataId)
			if err != nil {
				// print error and continue
				fmt.Printf("error processing verse scope: %s\n", err)
				continue
			}
		} else {
			if v.Scope == 4 {
				var tokenID, price sql.NullString
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT TOKENID, PRICE FROM LISTING_INFO WHERE ITEMDATAID = ? ORDER BY TIME DESC LIMIT 1", v.DataId).Scan(&tokenID, &price)
				if err != nil {
					continue
				} else {
					if tokenID.Valid {
						v.NftTokenID = tokenID.String
					}
					if price.Valid {
						v.Price = price.String
					}
				}
			}

			if v.Scope == 2 || v.Scope == 3 {
				v.NotInScope = v.Scope
			}
			if v.Scope == 5 {
				fmt.Printf("error processing verse scope: %s\n", err)
				continue
			}
		}

		verses = append(verses, v)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

func (r *resolver) SubscribedVerses(ctx context.Context, args subscribedVersesArgs) ([]*verse, error) {
	if args.UserDataId == nil {
		return nil, fmt.Errorf("UserDataId must be provided")
	}

	// Query to get a list of users the given user is following
	queryFollowing := "SELECT FOLLOWING FROM USER_FOLLOWING WHERE FOLLOWER = ? AND STATUS = 1"
	rowsFollowing, err := r.indexSvc.Db.QueryContext(ctx, queryFollowing, args.UserDataId)
	if err != nil {
		return nil, err
	}
	defer rowsFollowing.Close()

	// Collect the ids of the followed users
	var followedUsers []string
	for rowsFollowing.Next() {
		var followedUser string
		if err := rowsFollowing.Scan(&followedUser); err != nil {
			return nil, err
		}
		followedUsers = append(followedUsers, followedUser)
	}

	if len(followedUsers) == 0 {
		return []*verse{}, nil
	}

	// Prepare the query for the verses
	queryVerses := "SELECT * FROM VERSE WHERE OWNER IN (?" + strings.Repeat(", ?", len(followedUsers)-1) + ") AND STATUS != 2 ORDER BY CREATEDAT DESC"
	if args.Limit != nil {
		queryVerses += fmt.Sprintf(" LIMIT %d", *args.Limit)
	}
	if args.Offset != nil {
		queryVerses += fmt.Sprintf(" OFFSET %d", *args.Offset)
	}

	// Convert followedUsers to an interface slice for the db query
	followedUsersInterface := make([]interface{}, len(followedUsers))
	for i, v := range followedUsers {
		followedUsersInterface[i] = v
	}

	// Execute the query
	rowsVerses, err := r.indexSvc.Db.QueryContext(ctx, queryVerses, followedUsersInterface...)
	if err != nil {
		return nil, err
	}
	defer rowsVerses.Close()

	var verses []*verse
	for rowsVerses.Next() {
		v, err := verseFromRow(rowsVerses, ctx, r.indexSvc.Db, args.UserDataId)
		if err != nil {
			return nil, err
		}

		if args.UserDataId != nil {
			var count int
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWER = ? AND FOLLOWING = ?", args.UserDataId, v.Owner).Scan(&count)
			if err != nil {
				return nil, err
			}

			v.HasFollowedOwner = count > 0

			count = 0
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
			if err != nil {
				return nil, err
			}
			v.IsPaid = count > 0

			// Process verse scope
			v, err = processVerseScope(ctx, r.indexSvc.Db, v, *args.UserDataId)
			if err != nil {
				// print error and continue
				fmt.Printf("error processing verse scope: %s\n", err)
				continue
			}
		} else {
			if v.Scope == 2 || v.Scope == 3 || v.Scope == 4 {
				v.NotInScope = v.Scope
			}
			if v.Scope == 5 {
				fmt.Printf("error processing verse scope: %s\n", err)
				continue
			}
		}

		verses = append(verses, v)
	}

	if err = rowsVerses.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

func (r *resolver) LikedVerses(ctx context.Context, args likedVersesArgs) ([]*verse, error) {
	if args.UserDataId == nil {
		return nil, fmt.Errorf("UserDataId must be provided")
	}

	// Query to get a list of verses the given user has liked
	queryLiked := "SELECT VERSEID FROM VERSE_LIKE WHERE OWNER = ? AND STATUS = 1"
	rowsLiked, err := r.indexSvc.Db.QueryContext(ctx, queryLiked, args.UserDataId)
	if err != nil {
		return nil, err
	}
	defer rowsLiked.Close()

	// Collect the ids of the liked verses
	var likedVerses []string
	for rowsLiked.Next() {
		var likedVerse string
		if err := rowsLiked.Scan(&likedVerse); err != nil {
			return nil, err
		}
		likedVerses = append(likedVerses, likedVerse)
	}

	if len(likedVerses) == 0 {
		return []*verse{}, nil
	}

	// Prepare the query for the verses
	queryVerses := "SELECT * FROM VERSE WHERE DATAID IN (?" + strings.Repeat(", ?", len(likedVerses)-1) + ") AND STATUS != 2 ORDER BY CREATEDAT DESC"
	if args.Limit != nil {
		queryVerses += fmt.Sprintf(" LIMIT %d", *args.Limit)
	}
	if args.Offset != nil {
		queryVerses += fmt.Sprintf(" OFFSET %d", *args.Offset)
	}

	// Convert likedVerses to an interface slice for the db query
	likedVersesInterface := make([]interface{}, len(likedVerses))
	for i, v := range likedVerses {
		likedVersesInterface[i] = v
	}

	// Execute the query
	rowsVerses, err := r.indexSvc.Db.QueryContext(ctx, queryVerses, likedVersesInterface...)
	if err != nil {
		return nil, err
	}
	defer rowsVerses.Close()

	var verses []*verse
	for rowsVerses.Next() {
		v, err := verseFromRow(rowsVerses, ctx, r.indexSvc.Db, args.UserDataId)
		if err != nil {
			return nil, err
		}

		var count int
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWER = ? AND FOLLOWING = ?", args.UserDataId, v.Owner).Scan(&count)
		if err != nil {
			return nil, err
		}

		v.HasFollowedOwner = count > 0

		count = 0
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
		if err != nil {
			return nil, err
		}
		v.IsPaid = count > 0

		// Process verse scope
		v, err = processVerseScope(ctx, r.indexSvc.Db, v, *args.UserDataId)
		if err != nil {
			// print error and continue
			fmt.Printf("error processing verse scope: %s\n", err)
			continue
		}
		verses = append(verses, v)
	}

	if err = rowsVerses.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

func (r *resolver) VersesByIds(ctx context.Context, args struct {
	Ids          []string
	OnlyFollowed *bool
	UserDataId   *string
}) ([]*verse, error) {
	// Prepare the base query
	query := "SELECT * FROM VERSE"

	if len(args.Ids) <= 0 {
		// If no ids provided, return an empty result
		return []*verse{}, nil
	}

	placeholders := strings.Repeat("?,", len(args.Ids)-1) + "?"
	query += " WHERE DATAID IN (" + placeholders + ")"
	query += " AND STATUS !=2 ORDER BY CREATEDAT DESC"
	// convert []string to []interface{}
	argsInterface := make([]interface{}, len(args.Ids))
	for i, v := range args.Ids {
		argsInterface[i] = v
	}
	rows, err := r.indexSvc.Db.QueryContext(ctx, query, argsInterface...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var verses []*verse
	for rows.Next() {
		v, err := verseFromRow(rows, ctx, r.indexSvc.Db, args.UserDataId)
		if err != nil {
			return nil, err
		}

		if args.UserDataId != nil {
			var count int
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWER = ? AND FOLLOWING = ?", args.UserDataId, v.Owner).Scan(&count)
			if err != nil {
				return nil, err
			}

			v.HasFollowedOwner = count > 0
		}

		if args.OnlyFollowed == nil || !*args.OnlyFollowed || (*args.OnlyFollowed && v.HasFollowedOwner) {
			count := 0
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
			if err != nil {
				return nil, err
			}
			v.IsPaid = count > 0

			// Process verse scope
			v, err = processVerseScope(ctx, r.indexSvc.Db, v, *args.UserDataId)
			if err != nil {
				// print error and continue
				fmt.Printf("error processing verse scope: %s\n", err)
				continue
			}

			verses = append(verses, v)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

func verseFromRow(rowScanner interface{}, ctx context.Context, db *sql.DB, userDataId *string) (*verse, error) {
	var v verse

	var err error
	switch scanner := rowScanner.(type) {
	case *sql.Row:
		err = scanner.Scan(
			&v.CommitId,
			&v.DataId,
			&v.Alias,
			&v.CreatedAt,
			&v.FileIDs,
			&v.Owner,
			&v.Price,
			&v.Digest,
			&v.Scope,
			&v.Status,
			&v.NftTokenID,
			&v.FileType,
		)
	case *sql.Rows:
		err = scanner.Scan(
			&v.CommitId,
			&v.DataId,
			&v.Alias,
			&v.CreatedAt,
			&v.FileIDs,
			&v.Owner,
			&v.Price,
			&v.Digest,
			&v.Scope,
			&v.Status,
			&v.NftTokenID,
			&v.FileType,
		)
	default:
		return nil, fmt.Errorf("unsupported row scanner type")
	}

	if err != nil {
		return nil, err
	}

	// Fetch the comment count for the verse
	commentCountQuery := "SELECT COUNT(*) FROM verse_comment WHERE VerseID = ? AND STATUS != 2"
	err = db.QueryRowContext(ctx, commentCountQuery, v.DataId).Scan(&v.CommentCount)
	if err != nil {
		return nil, err
	}

	// Fetch the like count for the verse
	likeCountQuery := "SELECT COUNT(*) FROM verse_like WHERE STATUS = 1 AND VerseID = ?"
	err = db.QueryRowContext(ctx, likeCountQuery, v.DataId).Scan(&v.LikeCount)
	if err != nil {
		return nil, err
	}

	if userDataId != nil {
		// Query HasLiked from verse_like table
		var count int
		likeQuery := "SELECT COUNT(*) FROM verse_like WHERE STATUS = 1 AND  VerseID = ? AND OWNER = ?"
		err = db.QueryRowContext(ctx, likeQuery, v.DataId, userDataId).Scan(&count)
		fmt.Printf("sql: %s, params: %s\n", likeQuery, v.DataId, userDataId)

		if err != nil {
			fmt.Println("Error scanning row: ", err)
		}
		v.HasLiked = count > 0
	}

	// Query owner information from USER_PROFILE
	query := `SELECT ETHADDR, AVATAR, USERNAME, BIO FROM USER_PROFILE WHERE DATAID = ?`
	row := db.QueryRowContext(ctx, query, v.Owner)
	err = row.Scan(&v.OwnerEthAddr, &v.OwnerAvatar, &v.OwnerUsername, &v.OwnerBio)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No rows found, skipping.")
		} else {
			fmt.Println("Error scanning row: ", err)
		}
	}

	sixMonthsAgo := time.Now().AddDate(0, -6, 0).Unix() // 6 months ago in Unix time

	// Fetch the token ID and the price
	var tokenID, price sql.NullString
	err = db.QueryRowContext(ctx, "SELECT TOKENID, PRICE FROM LISTING_INFO WHERE ITEMDATAID = ? AND TIME >= ? ORDER BY TIME DESC LIMIT 1", v.Owner, sixMonthsAgo).Scan(&tokenID, &price)
	if err != nil && err != sql.ErrNoRows {
		// If the error is something other than 'no rows', return the error
		fmt.Printf("Error fetching token ID and price: %v\n", err)
	} else {
		if tokenID.Valid {
			v.OwnerNftTokenID = tokenID.String
		}
		// If the price exists, check if verse is already purchased
		if price.Valid {
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.Owner, userDataId).Scan(&count)
			if err != nil {
				return nil, err
			}
			if count == 0 { // not purchased yet
				v.PayToFollow = price.String
			}
		}
	}

	return &v, nil
}

func processVerseScope(ctx context.Context, db *sql.DB, v *verse, userDataId string) (*verse, error) {
	// Check if the user follows the owner of the verse
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWER = ? AND FOLLOWING = ?", userDataId, v.Owner).Scan(&count)
	if err != nil {
		return nil, err
	}
	v.HasFollowedOwner = count > 0

	// Fetching the NftTokenID and price for the verse if v.Scope = 4
	if v.Scope == 4 {
		var tokenID, price sql.NullString
		err = db.QueryRowContext(ctx, "SELECT TOKENID, PRICE FROM LISTING_INFO WHERE ITEMDATAID = ? ORDER BY TIME DESC LIMIT 1", v.DataId).Scan(&tokenID, &price)
		if err != nil {
			if err == sql.ErrNoRows {
				// No rows in the result set
				// Check if v.CreatedAt is older than 5 minutes
				currentTime := types.Uint64(time.Now().UnixNano() / 1e6) // Convert to Unix milliseconds
				if currentTime-v.CreatedAt > 5*60*1000 {
					v.NftTokenID = "failed"
				}
			} else {
				// If the error is something other than 'no rows', return the error
				fmt.Printf("Error fetching token ID and price: %v\n", err)
			}

			if userDataId != v.Owner {
				return nil, fmt.Errorf("verse %s is not ready", v.DataId)
			}
		} else {
			if tokenID.Valid {
				v.NftTokenID = tokenID.String
			}
			if price.Valid {
				v.Price = price.String
			}
		}
	}

	// Check verse scope conditions and modify the verse accordingly
	if userDataId != v.Owner {
		switch v.Scope {
		case 2:
			if !v.HasFollowedOwner {
				v.NotInScope = 2
			}
		case 3:
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM USER_FOLLOWING WHERE STATUS =1 AND FOLLOWING = ? AND FOLLOWER = ?", userDataId, v.Owner).Scan(&count)
			if err != nil {
				return nil, err
			}
			if count == 0 {
				v.NotInScope = 3
			}
		case 4:
			if !v.IsPaid {
				v.NotInScope = 4
			}
		case 5:
			return nil, fmt.Errorf("verse is private")
		}
	}

	return v, nil
}

func (v *verse) ID() graphql.ID {
	return graphql.ID(v.CommitId)
}