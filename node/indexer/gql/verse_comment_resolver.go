package gql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"sao-node/node/indexer/gql/types"
)

type verseComment struct {
	CommitId      string       `json:"CommitId"`
	DataId        string       `json:"DataId"`
	Alias         string       `json:"Alias"`
	CreatedAt     types.Uint64 `json:"CreatedAt"`
	UpdatedAt     types.Uint64 `json:"UpdatedAt"`
	VerseID       string       `json:"VerseID"`
	Owner         string       `json:"Owner"`
	Status		int32        `json:"Status"`
	Comment       string       `json:"Comment"`
	Parent *verseComment `json:"Parent"`
	LikeCount     int32        `json:"LikeCount"`
	HasLiked      bool         `json:"HasLiked"`
	OwnerEthAddr  string       `json:"OwnerEthAddr"`
	OwnerAvatar   string       `json:"OwnerAvatar"`
	OwnerUsername string       `json:"OwnerUsername"`
	OwnerBio      string       `json:"OwnerBio"`
}

type verseCommentsArgs struct {
	VerseID    string
	Limit      *int32
	Offset     *int32
	UserDataId *string
}

func (r *resolver) VerseComments(ctx context.Context, args verseCommentsArgs) ([]*verseComment, error) {
	limit := 10
	offset := 0

	if args.Limit != nil {
		limit = int(*args.Limit)
	}

	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, `
			SELECT VC.*, UP.ETHADDR, UP.AVATAR, UP.USERNAME, UP.BIO 
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
		SELECT VC.*, UP.ETHADDR, UP.AVATAR, UP.USERNAME, UP.BIO 
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
