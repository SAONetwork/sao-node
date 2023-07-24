package storverse

import (
	"fmt"
)

type ListingInfo struct {
	ID         string
	Price      string
	TokenId    string
	ItemDataId string
	ChainType  string
	Time       int
	Type       string
	CommitID   string
	DataID     string
	Alias      string
}

type ListingInfoInsertionStrategy struct{}

func (l ListingInfo) InsertValues() string {
	return fmt.Sprintf("('%s','%s','%s','%s','%s','%s','%s',%d, '%s')",
		l.CommitID, l.DataID, l.Alias, l.Price, l.TokenId, l.ItemDataId, l.ChainType, l.Time, l.Type)
}

func (s ListingInfoInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(ListingInfo)
}

func (s ListingInfoInsertionStrategy) TableName() string {
	return "LISTING_INFO"
}