package storverse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Verse struct {
	ID         *string  `json:"id,omitempty"`
	CreatedAt  int64    `json:"createdAt"`
	FileIDs    []string `json:"fileIds"`
	Owner      string   `json:"owner"`
	Price      string    `json:"price"`
	Digest     string   `json:"digest"`
	Scope      int      `json:"scope"`
	Status     int      `json:"status"`
	NftTokenID string   `json:"nftTokenId"`
	FileType   string   `json:"fileType"`
	CommitID   string
	DataID     string
	Alias      string
}

type VerseInsertionStrategy struct{}

func (v Verse) InsertValues() string {
	// Serialize the FileIDs into a JSON string
	fileIDsJSON, err := json.Marshal(v.FileIDs)
	if err != nil {
		// handle error, log the error
		fmt.Println(err)
	}

	price := 0.0
	if v.Price != "" {
		var err error
		price, err = strconv.ParseFloat(v.Price, 64)
		if err != nil {
			// handle error, log the error
			fmt.Println(err)
		}
	}

	return fmt.Sprintf("('%s','%s','%s',%d,'%s','%s',%f,'%s',%d,%d,'%s','%s')",
		v.CommitID, v.DataID, v.Alias, v.CreatedAt, string(fileIDsJSON), v.Owner, price, EscapeSingleQuotes(v.Digest), v.Scope, v.Status, v.NftTokenID, v.FileType)
}

func (s VerseInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(Verse)
}

func (s VerseInsertionStrategy) TableName() string {
	return "VERSE"
}

func EscapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func GetVerseOwnerAndDigestAndFiletypeByVerseID(db *sql.DB, verseID string) (string, string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var owner string
	var digest string
	var filetype string
	err := db.QueryRowContext(ctx, "SELECT OWNER, DIGEST, FILETYPE FROM VERSE WHERE DATAID = ?", verseID).Scan(&owner, &digest, &filetype)
	if err != nil {
		return "", "", "", err
	}

	return owner, digest, filetype, nil
}
