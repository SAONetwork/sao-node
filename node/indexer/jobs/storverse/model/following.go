package storverse

import (
	"fmt"
)

type UserFollowing struct {
	ID        string `json:"id,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	ExpiredAt int64  `json:"expiredAt"`
	Follower  string `json:"follower"`
	Following string `json:"following"`
	Status    string    `json:"status"`
	CommitID  string
	DataID    string
	Alias 	  string
}

type UserFollowingInsertionStrategy struct{}

func (uf UserFollowing) InsertValues() string {
	return fmt.Sprintf("('%s','%s','%s', %d, %d, %d, '%s', '%s', %s)",
		uf.CommitID, uf.DataID, uf.Alias, uf.CreatedAt, uf.UpdatedAt, uf.ExpiredAt, uf.Follower, uf.Following, uf.Status)
}

func (s UserFollowingInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(UserFollowing)
}

func (s UserFollowingInsertionStrategy) TableName() string {
	return "USER_FOLLOWING"
}