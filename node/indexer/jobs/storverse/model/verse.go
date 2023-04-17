package storverse

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Verse struct {
	ID         *string  `json:"id,omitempty"`
	CreatedAt  int64    `json:"createdAt"`
	FileIDs    []string `json:"fileIds"`
	Owner      string   `json:"owner"`
	Price      string   `json:"price"`
	Digest     string   `json:"digest"`
	Scope      string   `json:"scope"`
	Status     string   `json:"status"`
	NftTokenID string   `json:"nftTokenId"`
	CommitID   string
	DataID     string
}

func (v Verse) InsertValues() string {
	price, err := strconv.ParseFloat(v.Price, 64)
	if err != nil {
		// handle error
	}

	scope, err := strconv.Atoi(v.Scope)
	if err != nil {
		// handle error
	}

	status, err := strconv.Atoi(v.Status)
	if err != nil {
		// handle error
	}

	// Serialize the FileIDs into a JSON string
	fileIDsJSON, err := json.Marshal(v.FileIDs)
	if err != nil {
		// handle error
	}

	return fmt.Sprintf("('%s','%s',%d,'%s','%s',%.2f,'%s',%d,%d,'%s')",
		v.CommitID, v.DataID, v.CreatedAt, string(fileIDsJSON), v.Owner, price, v.Digest, scope, status, v.NftTokenID)

}
