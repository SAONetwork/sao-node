package storverse

import (
	"fmt"
)

type PurchaseOrder struct {
	ID          string
	OrderID     uint64
	ItemDataID  string
	BuyerDataID string
	OrderTxHash string
	ChainType   string
	Price       string
	Time        uint64
	Type        int32
	ExpireTime  uint64
	CommitID    string
	DataID      string
	Alias       string
}

func (p PurchaseOrder) InsertValues() string {
	return fmt.Sprintf("('%s','%s','%s', %d, '%s','%s','%s','%s','%s', %d, %d, '%d')",
		p.CommitID, p.DataID, p.Alias, p.OrderID, p.ItemDataID, p.BuyerDataID, p.OrderTxHash, p.ChainType, p.Price, p.Time, p.Type, p.ExpireTime)
}
