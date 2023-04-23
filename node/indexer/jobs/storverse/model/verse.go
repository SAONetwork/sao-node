package storverse

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Verse struct {
	ID         *string  `json:"id,omitempty"`
	CreatedAt  int64    `json:"createdAt"`
	FileIDs    []string `json:"fileIds"`
	Owner      string   `json:"owner"`
	Price      string   `json:"price"`
	Digest     string   `json:"digest"`
	Scope      int      `json:"scope"`
	Status     int      `json:"status"`
	NftTokenID string   `json:"nftTokenId"`
	CommitID   string
	DataID     string
	Alias      string
}

func (v Verse) InsertValues() string {
	price, err := strconv.ParseFloat(v.Price, 64)
	if err != nil {
		// handle error
	}

	// Serialize the FileIDs into a JSON string
	fileIDsJSON, err := json.Marshal(v.FileIDs)
	if err != nil {
		// handle error
	}

	return fmt.Sprintf("('%s','%s','%s',%d,'%s','%s',%.2f,'%s',%d,%d,'%s')",
		v.CommitID, v.DataID, v.Alias, v.CreatedAt, string(fileIDsJSON), v.Owner, price, escapeSingleQuotes(v.Digest), v.Scope, v.Status, v.NftTokenID)
}

func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
