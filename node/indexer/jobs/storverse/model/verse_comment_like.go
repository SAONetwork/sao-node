package storverse

import (
	"fmt"
)

type VerseCommentLike struct {
	ID        string `json:"id,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	CommentId string `json:"commentId"`
	Status    int    `json:"status"`
	Owner     string `json:"owner"`
	CommitID  string
	DataID    string
	Alias     string
}

type VerseCommentLikeInsertionStrategy struct{}

func (vcl VerseCommentLike) InsertValues() string {
	return fmt.Sprintf("('%s', '%s', '%s', %d, %d, '%s', %d, '%s')",
		vcl.CommitID, vcl.DataID, vcl.Alias, vcl.CreatedAt, vcl.UpdatedAt, vcl.CommentId, vcl.Status, vcl.Owner)
}

func (s VerseCommentLikeInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(VerseCommentLike)
}

func (s VerseCommentLikeInsertionStrategy) TableName() string {
	return "VERSE_COMMENT_LIKE"
}
