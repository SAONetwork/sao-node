package gql

import (
	"context"
	"sao-node/node/indexer/gql/types"
	"strconv"
)

type NotificationsInfo struct {
	Items        []*Notification    `json:"items"`
	TotalCount   int32              `json:"totalCount"`
	UnreadCounts []*UnreadCountInfo `json:"unreadCounts"`
}

type UnreadCountInfo struct {
	MessageType string `json:"messageType"`
	UnreadCount int32  `json:"unreadCount"`
}

type Notification struct {
	BaseDataID  string       `json:"baseDataId"`
	CreatedAt   types.Uint64 `json:"createdAt"`
	UpdatedAt   types.Uint64 `json:"updatedAt"`
	FromUser    string       `json:"fromUser"`
	ToUser      string       `json:"toUser"`
	MessageType int32        `json:"messageType"`
	Message     string       `json:"message"`
	Status      string       `json:"status"`
}

type notificationsArgs struct {
	MessageType string
	ToUser      string
	Limit       *int32
	Offset      *int32
}

func (r *resolver) Notifications(ctx context.Context, args notificationsArgs) (*NotificationsInfo, error) {
	limit := 10
	offset := 0

	if args.Limit != nil {
		limit = int(*args.Limit)
	}

	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	messageType, err := strconv.Atoi(args.MessageType)
	if err != nil {
		return nil, err
	}

	// Fetch Notification items
	rows, err := r.indexSvc.Db.QueryContext(ctx, `
		SELECT 
			CASE
				WHEN n.MessageType = 2 THEN COALESCE((SELECT ITEMDATAID FROM PURCHASE_ORDER WHERE DATAID = n.BaseDataID), n.BaseDataID)
				WHEN n.MessageType IN (4, 7) THEN COALESCE((SELECT VERSEID FROM VERSE_COMMENT WHERE DATAID = n.BaseDataID), n.BaseDataID)
				WHEN n.MessageType = 5 THEN COALESCE((SELECT VERSEID FROM VERSE_LIKE WHERE DATAID = n.BaseDataID), n.BaseDataID)
				WHEN n.MessageType = 6 THEN COALESCE((SELECT VERSEID FROM VERSE_COMMENT WHERE DATAID = (SELECT COMMENTID FROM VERSE_COMMENT_LIKE WHERE DATAID = n.BaseDataID)), n.BaseDataID)
				ELSE n.BaseDataID
			END as BASEDATAID,
			n.CreatedAt,
			n.UpdatedAt,
			n.Message,
			n.Status,
			n.MessageType,
			n.FromUser,
			n.ToUser
		FROM NOTIFICATION n
		WHERE 
			CASE 
				WHEN ? = 2 THEN n.MessageType IN (2, 3)
				WHEN ? = 4 THEN n.MessageType IN (4, 7)
				WHEN ? = 5 THEN n.MessageType IN (5, 6)
				ELSE n.MessageType = ?
			END 
			AND n.ToUser = ? 
		ORDER BY n.CreatedAt DESC LIMIT ? OFFSET ?
	`, messageType, messageType, messageType, messageType, args.ToUser, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Notification
	for rows.Next() {
		var n Notification
		err := rows.Scan(
			&n.BaseDataID,
			&n.CreatedAt,
			&n.UpdatedAt,
			&n.Message,
			&n.Status,
			&n.MessageType,
			&n.FromUser,
			&n.ToUser,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, &n)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	// Fetch totalCount
	var totalCount int32
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM NOTIFICATION WHERE MESSAGETYPE = ? AND TOUSER = ?", args.MessageType, args.ToUser).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	// Fetch UnreadCounts
	unreadCountsRows, err := r.indexSvc.Db.QueryContext(ctx, `
        SELECT 
            CASE 
                WHEN MESSAGETYPE IN (2, 3) THEN 2
                WHEN MESSAGETYPE IN (4, 7) THEN 4
                WHEN MESSAGETYPE IN (5, 6) THEN 5
                ELSE MESSAGETYPE
            END as MessageType,
            COUNT(*) as UnreadCount 
        FROM NOTIFICATION 
        WHERE TOUSER = ? AND STATUS = 0 
        GROUP BY MessageType`, args.ToUser)

	if err != nil {
		return nil, err
	}
	defer unreadCountsRows.Close()

	var unreadCounts []*UnreadCountInfo
	for unreadCountsRows.Next() {
		var uc UnreadCountInfo
		err := unreadCountsRows.Scan(&uc.MessageType, &uc.UnreadCount)
		if err != nil {
			return nil, err
		}
		unreadCounts = append(unreadCounts, &uc)
	}
	err = unreadCountsRows.Err()
	if err != nil {
		return nil, err
	}

	// Return a NotificationsInfo object containing the fetched data
	return &NotificationsInfo{
		Items:        items,
		TotalCount:   totalCount,
		UnreadCounts: unreadCounts,
	}, nil
}
