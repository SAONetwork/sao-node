package storverse

import (
	"fmt"
)

type VerseComment struct {
	ID        string `json:"id,omitempty"`
	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
	Comment   string `json:"comment"`
	ParentId  string `json:"parentId"`
	VerseId   string `json:"verseId"`
	Owner     string `json:"owner"`
	CommitID  string
	DataID    string
	Alias 	  string
}

type VerseCommentInsertionStrategy struct{}

func (vc VerseComment) InsertValues() string {
	return fmt.Sprintf("('%s', '%s','%s', '%d', '%d', '%s', '%s', '%s', '%s')",
		vc.CommitID, vc.DataID, vc.Alias, vc.CreatedAt, vc.UpdatedAt, vc.Comment, vc.ParentId, vc.VerseId, vc.Owner)
}

func (s VerseCommentInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(VerseComment)
}

func (s VerseCommentInsertionStrategy) TableName() string {
	return "VERSE_COMMENT"
}
