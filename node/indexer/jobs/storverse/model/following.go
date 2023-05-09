package storverse

import (
	"context"
	"database/sql"
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

func UpdateUserFollowingStatus(ctx context.Context, db *sql.DB) (int64, error) {
	updateQuery := `UPDATE USER_FOLLOWING
                SET status = 1
                WHERE EXISTS (
                  SELECT 1 FROM PURCHASE_ORDER
                  WHERE PURCHASE_ORDER.TYPE=2 AND PURCHASE_ORDER.ITEMDATAID = USER_FOLLOWING.FOLLOWING
                  AND PURCHASE_ORDER.BUYERDATAID = USER_FOLLOWING.FOLLOWER
                );`

	result, err := db.ExecContext(ctx, updateQuery)
	if err != nil {
		return 0, fmt.Errorf("Error updating USER_FOLLOWING records: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("Error getting the number of updated rows: %v", err)
	}

	return rowsAffected, nil
}