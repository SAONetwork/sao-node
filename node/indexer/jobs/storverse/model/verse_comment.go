package storverse

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"
)

type VerseComment struct {
	ID        string `json:"id,omitempty"`
	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
	Comment   string `json:"comment"`
	ParentId  string `json:"parentId"`
	VerseId   string `json:"verseId"`
	Owner     string `json:"owner"`
	Status    int    `json:"status"`
	CommitID  string
	DataID    string
	Alias 	  string
}

type VerseCommentInsertionStrategy struct{}

func (vc VerseComment) InsertValues() string {
	decodedComment, err := url.QueryUnescape(vc.Comment)
	if err != nil {
		// log
		fmt.Println(err)
	}
	return fmt.Sprintf("('%s', '%s','%s', '%d', '%d', '%s', '%s', '%s', '%s', %d)",
		vc.CommitID, vc.DataID, vc.Alias, vc.CreatedAt, vc.UpdatedAt, decodedComment, vc.ParentId, vc.VerseId, vc.Owner, vc.Status)
}

func (s VerseCommentInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(VerseComment)
}

func (s VerseCommentInsertionStrategy) TableName() string {
	return "VERSE_COMMENT"
}

func GetVerseCommentOwnerByCommentID(db *sql.DB, commentID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var owner string
	err := db.QueryRowContext(ctx, "SELECT OWNER FROM VERSE_COMMENT WHERE DATAID = ?", commentID).Scan(&owner)
	if err != nil {
		return "", err
	}

	return owner, nil
}