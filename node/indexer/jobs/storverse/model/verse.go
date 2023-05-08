package storverse

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Verse struct {
	ID         *string  `json:"id,omitempty"`
	CreatedAt  int64    `json:"createdAt"`
	FileIDs    []string `json:"fileIds"`
	Owner      string   `json:"owner"`
	Price      int64   `json:"price"`
	Digest     string   `json:"digest"`
	Scope      int      `json:"scope"`
	Status     int      `json:"status"`
	NftTokenID string   `json:"nftTokenId"`
	CommitID   string
	DataID     string
	Alias      string
}

type VerseInsertionStrategy struct{}

func (v Verse) InsertValues() string {
	// Serialize the FileIDs into a JSON string
	fileIDsJSON, err := json.Marshal(v.FileIDs)
	if err != nil {
		// handle error
	}

	return fmt.Sprintf("('%s','%s','%s',%d,'%s','%s',%d,'%s',%d,%d,'%s')",
		v.CommitID, v.DataID, v.Alias, v.CreatedAt, string(fileIDsJSON), v.Owner, v.Price, escapeSingleQuotes(v.Digest), v.Scope, v.Status, v.NftTokenID)
}

func (s VerseInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(Verse)
}

func (s VerseInsertionStrategy) TableName() string {
	return "VERSE"
}

func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
