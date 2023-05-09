package storverse

import (
	"fmt"
)

type ReadNotifications struct {
	Time        int64  `json:"time"`
	Owner       string `json:"owner"`
	MessageType int    `json:"messageType"`
	CommitID    string
	DataID      string
	Alias       string
}

type ReadNotificationsInsertionStrategy struct{}

func (rn ReadNotifications) InsertValues() string {
	return fmt.Sprintf("('%s', '%s','%s', '%d', '%s', '%d')",
		rn.CommitID, rn.DataID, rn.Alias, rn.Time, rn.Owner, rn.MessageType)
}

func (s ReadNotificationsInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(ReadNotifications)
}

func (s ReadNotificationsInsertionStrategy) TableName() string {
	return "READ_NOTIFICATIONS"
}
