package storverse

import (
	"fmt"
)

type ListingInfo struct {
	ID       string
	Price    string
	TokenId  string
	ItemDataId string
	ChainType string
	Time     int
	CommitID string
	DataID   string
	Alias    string
}

func (l ListingInfo) InsertValues() string {
	return fmt.Sprintf("('%s','%s','%s','%s','%s','%s','%s',%d)",
		l.CommitID, l.DataID, l.Alias, l.Price, l.TokenId, l.ItemDataId, l.ChainType, l.Time)
}
