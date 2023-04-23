package storverse

import (
	"fmt"
	"strings"
)

type UserProfile struct {
	ID              string
	CreatedAt       int
	UpdatedAt       int
	DID             string
	EthAddr         string
	Avatar          string
	Username        string
	FollowingCount  int
	Twitter         string
	Youtube         string
	Bio             string
	Banner          string
	FollowingDataId []string
	CommitID        string
	DataID          string
	Alias           string
}

func (u UserProfile) InsertValues() string {
	followingDataID := ""
	if len(u.FollowingDataId) > 0 {
		followingDataID = strings.Join(u.FollowingDataId, ",")
	}

	return fmt.Sprintf("('%s','%s','%s',%d,%d,'%s','%s','%s','%s',%d,'%s','%s','%s','%s','%s')",
		u.CommitID, u.DataID, u.Alias, u.CreatedAt, u.UpdatedAt, u.DID, u.EthAddr, u.Avatar, u.Username,
		u.FollowingCount, u.Twitter, u.Youtube, u.Bio, u.Banner, followingDataID)
}
