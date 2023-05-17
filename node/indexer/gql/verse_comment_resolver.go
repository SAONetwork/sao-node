package gql

import (
	"context"
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
	Comment       string       `json:"Comment"`
	ParentID      string       `json:"ParentID"`
	LikeCount     int32        `json:"LikeCount"`
	HasLiked      bool         `json:"HasLiked"`
	OwnerEthAddr  string       `json:"OwnerEthAddr"`
	OwnerAvatar   string       `json:"OwnerAvatar"`
	OwnerUsername string       `json:"OwnerUsername"`
	OwnerBio      string       `json:"OwnerBio"`
}

type verseCommentsArgs struct {
	VerseID string
	Limit   *int32
	Offset  *int32
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
			WHERE VC.VERSEID = ? 
			LIMIT ? OFFSET ?`,
		args.VerseID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*verseComment
	for rows.Next() {
		var c verseComment
		err := rows.Scan(
			&c.CommitId,
			&c.DataId,
			&c.Alias,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.Comment,
			&c.ParentID,
			&c.VerseID,
			&c.Owner,
			&c.OwnerEthAddr,
			&c.OwnerAvatar,
			&c.OwnerUsername,
			&c.OwnerBio,
		)
		if err != nil {
			return nil, err
		}

		// Add a query to get the like count from the VERSE_COMMENT_LIKE table
		err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM VERSE_COMMENT_LIKE WHERE COMMENTID = ? AND STATUS = 1", c.DataId).Scan(&c.LikeCount)
		if err != nil {
			return nil, err
		}

		comments = append(comments, &c)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (vc *verseComment) ID() graphql.ID {
	return graphql.ID(vc.CommitId)
}
