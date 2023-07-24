package gql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
	"strconv"
)

type verseComment struct {
	CommitId      string        `json:"CommitId"`
	DataId        string        `json:"DataId"`
	Alias         string        `json:"Alias"`
	CreatedAt     types.Uint64  `json:"CreatedAt"`
	UpdatedAt     types.Uint64  `json:"UpdatedAt"`
	VerseID       string        `json:"VerseID"`
	Owner         string        `json:"Owner"`
	Status        int32         `json:"Status"`
	Comment       string        `json:"Comment"`
	Parent        *verseComment `json:"Parent"`
	LikeCount     int32         `json:"LikeCount"`
	HasLiked      bool          `json:"HasLiked"`
	OwnerEthAddr  string        `json:"OwnerEthAddr"`
	OwnerAvatar   string        `json:"OwnerAvatar"`
	OwnerUsername string        `json:"OwnerUsername"`
	OwnerBio      string        `json:"OwnerBio"`
}

type verseCommentsArgs struct {
	VerseID    string
	Limit      *int32
	Offset     *int32
	UserDataId *string
}

func (r *resolver) VerseComments(ctx context.Context, args verseCommentsArgs) ([]*verseComment, error) {
	claims, ok := ctx.Value("claims").(string)
	// If UserDataId is not nil, require it to match the claims
	if args.UserDataId != nil && (!ok || claims != *args.UserDataId) {
		return nil, errors.New("Unauthorized")
	}

	limit := 10
	offset := 0

	if args.Limit != nil {
		limit = int(*args.Limit)
	}

	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	// Find verse by fileInfo ID
	var v verse
	err := r.indexSvc.Db.QueryRowContext(ctx, "SELECT * FROM VERSE WHERE DATAID = ?", args.VerseID).Scan(
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

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		return nil, errors.New("verse not found")
	}

	// If verse price is greater than 0, check if there's a PurchaseOrder record with ItemDataID = verse.DATAID and BuyerDataID = userDataId
	if v.Price != "" {
		price, err := strconv.ParseFloat(v.Price, 64)
		if err != nil {
			return nil, err
		}

		if price > 0 {
			var count int
			if args.UserDataId != nil {
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
			}
			if err != nil {
				return nil, err
			}

			v.IsPaid = count > 0
		}
	}

	if args.UserDataId != nil {
		// If verse price is greater than 0, check if there's a PurchaseOrder record with ItemDataID = verse.DATAID and BuyerDataID = userDataId
		if v.Price != "" {
			price, err := strconv.ParseFloat(v.Price, 64)
			if err != nil {
				return nil, err
			}

			if price > 0 {
				var count int
				err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM PURCHASE_ORDER WHERE ITEMDATAID = ? AND BUYERDATAID = ?", v.DataId, *args.UserDataId).Scan(&count)
				if err != nil {
					return nil, err
				}

				if count == 0 {
					return nil, errors.New("the verse is charged and not paid yet")
				}
				v.IsPaid = count > 0
			}
		}

		// Process verse scope
		_, err = processVerseScope(ctx, r.indexSvc.Db, &v, *args.UserDataId)
		if err != nil {
			return nil, err
		}

		if v.NotInScope > 1 {
			// verse is not accessible, return
			return nil, errors.New("you are not authorized to access the file")
		}
	} else {
		if v.Scope == 2 || v.Scope == 3 || v.Scope == 4 {
			return nil, errors.New("you are not authorized to access the comments")
		}
		if v.Scope == 5 {
			return nil, errors.New("the verse is private")
		}
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, `
			SELECT VC.*, COALESCE(UP.ETHADDR, ''), COALESCE(UP.AVATAR, ''), COALESCE(UP.USERNAME, ''), COALESCE(UP.BIO, '') 
			FROM VERSE_COMMENT VC
			LEFT JOIN USER_PROFILE UP ON VC.OWNER = UP.DATAID
			WHERE VC.STATUS !=2 AND VC.VERSEID = ? ORDER BY VC.CREATEDAT DESC
			LIMIT ? OFFSET ?`,
		args.VerseID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*verseComment
	for rows.Next() {
		var parentID string
		var c verseComment
		err := rows.Scan(
			&c.CommitId,
			&c.DataId,
			&c.Alias,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.Comment,
			&parentID,
			&c.VerseID,
			&c.Owner,
			&c.Status,
			&c.OwnerEthAddr,
			&c.OwnerAvatar,
			&c.OwnerUsername,
			&c.OwnerBio,
		)
		if err != nil {
			return nil, err
		}

		if parentID != "" {
			c.Parent, err = r.getParentCommentByID(ctx, parentID)
			if err != nil {
				fmt.Printf("Error getting parent comment: %s\n", err.Error())
			}
		}

		// Get the like count
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM VERSE_COMMENT_LIKE WHERE COMMENTID = ? AND STATUS = 1", c.DataId).Scan(&c.LikeCount)
		if err != nil {
			return nil, err
		}

		// Check if the current user has liked this comment
		if args.UserDataId != nil {
			var likeStatus int
			err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT STATUS FROM VERSE_COMMENT_LIKE WHERE COMMENTID = ? AND OWNER = ? AND STATUS != 2", c.DataId, *args.UserDataId).Scan(&likeStatus)
			if err != nil && err != sql.ErrNoRows {
				fmt.Printf("Error checking if user has liked comment: %s\n", err.Error())
			}
			// If a like exists (status = 1), set HasLiked to true
			c.HasLiked = likeStatus == 1
		}

		comments = append(comments, &c)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (r *resolver) getParentCommentByID(ctx context.Context, id string) (*verseComment, error) {
	row := r.indexSvc.Db.QueryRowContext(ctx, `
		SELECT VC.*, COALESCE(UP.ETHADDR, ''), COALESCE(UP.AVATAR, ''), COALESCE(UP.USERNAME, ''), COALESCE(UP.BIO, '') 
		FROM VERSE_COMMENT VC
		LEFT JOIN USER_PROFILE UP ON VC.OWNER = UP.DATAID
		WHERE VC.DATAID = ?`, id)

	var c verseComment
	var parentID string
	err := row.Scan(
		&c.CommitId,
		&c.DataId,
		&c.Alias,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.Comment,
		&parentID,
		&c.VerseID,
		&c.Owner,
		&c.Status,
		&c.OwnerEthAddr,
		&c.OwnerAvatar,
		&c.OwnerUsername,
		&c.OwnerBio,
	)

	if c.Status == 2 {
		c.Comment = "This comment has been deleted"
	}

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (vc *verseComment) ID() graphql.ID {
	return graphql.ID(vc.CommitId)
}
