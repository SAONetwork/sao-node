package storverse

import (
	"fmt"
)

type VerseLike struct {
	ID        string `json:"id,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	VerseId   string `json:"verseId"`
	Status    int    `json:"status"`
	Owner     string `json:"owner"`
	CommitID  string
	DataID    string
	Alias     string
}

type VerseLikeInsertionStrategy struct{}

func (vl VerseLike) InsertValues() string {
	return fmt.Sprintf("('%s', '%s', '%s', %d, %d, '%s', %d, '%s')",
		vl.CommitID, vl.DataID, vl.Alias, vl.CreatedAt, vl.UpdatedAt, vl.VerseId, vl.Status, vl.Owner)
}

func (s VerseLikeInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(VerseLike)
}

func (s VerseLikeInsertionStrategy) TableName() string {
	return "VERSE_LIKE"
}
