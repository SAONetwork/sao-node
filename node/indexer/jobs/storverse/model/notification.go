package storverse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type Notification struct {
	BaseDataID    string `json:"dataId"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
	MessageType	int    `json:"messageType"`
	FromUser  string `json:"fromUser"`
	ToUser    string `json:"toUser"`
}

type NotificationInsertionStrategy struct{}

func (n Notification) InsertValues() string {
	return fmt.Sprintf("('%s', '%d', '%d', '%s', '%d','%d', '%s', '%s')",
		n.BaseDataID, n.CreatedAt, n.UpdatedAt, n.Message, n.Status, n.MessageType, n.FromUser, n.ToUser)
}

func (s NotificationInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(Notification)
}

func (s NotificationInsertionStrategy) TableName() string {
	return "NOTIFICATION"
}

func CreateNotification(db *sql.DB, record BatchInserter) (*Notification, error, bool) {
	var fromUser, toUser, baseDataID, message string
	var messageType int
	var notificationTime int64

	switch r := record.(type) {
	case UserFollowing:
		if r.Status == 0 {
			return nil, errors.New("cancel follow, no need to notification"), false
		}
		fromUser = r.Follower
		toUser = r.Following
		baseDataID = r.DataID
		messageType = 1
		notificationTime = r.CreatedAt
	case PurchaseOrder:
		fromUser = r.BuyerDataID
		// if r.Type = 1, it means the purchase order is for a verse, so fetch the verse owner
		if r.Type == 1 {
			verseOwner, digest, fileType, err := GetVerseOwnerAndDigestAndFiletypeByVerseID(db, r.ItemDataID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, errors.New("verse not found"), true
				}
				return nil, err, false
			}
			toUser = verseOwner
			messageType = 2
			if digest != "" {
				digest = truncateMessageContent(digest)
				message = digest
			} else {
				message = fmt.Sprintf("[%s]", fileType)
			}
		} else {
			// if r.Type = 2, it means the purchase order is for a user, so fetch the user
			toUser = r.ItemDataID
			messageType = 3
			message = r.Price
		}
		baseDataID = r.DataID
		notificationTime = int64(r.Time)
	case VerseComment:
		fromUser = r.Owner
		var recipient, messageContent, fileType string
		var err error

		if r.ParentId == "" {
			// Fetch the verse owner
			recipient, messageContent, fileType, err = GetVerseOwnerAndDigestAndFiletypeByVerseID(db, r.VerseId)
			if messageContent == "" {
				messageContent = fmt.Sprintf("[%s]", fileType)
			}
			messageType = 4
		} else {
			// Fetch the parent comment
			messageContent, recipient, err = GetVerseCommentAndOwnerByCommentID(db, r.ParentId)
			messageType = 7
		}

		if err != nil {
			return nil, err, true
		}

		toUser = recipient
		baseDataID = r.DataID
		notificationTime = r.CreatedAt
		messageContent = truncateMessageContent(messageContent)
		message = messageContent
	case VerseLike:
		// Add similar logic for VerseLike
		fromUser = r.Owner
		// Fetch the verse owner
		verseOwner, digest, fileType, err := GetVerseOwnerAndDigestAndFiletypeByVerseID(db, r.VerseId)
		if err != nil {
			return nil, err, true
		}
		toUser = verseOwner
		baseDataID = r.DataID
		messageType = 5
		notificationTime = r.CreatedAt
		if digest != "" {
			digest = truncateMessageContent(digest)
			message = digest
		} else {
			message = fmt.Sprintf("[%s]", fileType)
		}
	case VerseCommentLike:
		fromUser = r.Owner
		// Fetch the verse comment owner
		commentContent, verseCommentOwner, err := GetVerseCommentAndOwnerByCommentID(db, r.CommentId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("verse comment not found"), true
			}
			return nil, err, false
		}
		toUser = verseCommentOwner
		baseDataID = r.DataID
		messageType = 6
		notificationTime = r.CreatedAt
		commentContent = truncateMessageContent(commentContent)
		message = commentContent
	default:
		return nil, errors.New("unsupported record type for creating notifications"), false
	}

	// Create a new notification
	notification := &Notification{
		BaseDataID:  baseDataID,
		CreatedAt:   notificationTime,
		UpdatedAt:   notificationTime,
		Message:     message,
		Status:      0,
		MessageType: messageType,
		FromUser:    fromUser,
		ToUser:      toUser,
	}

	return notification, nil, false
}

func UpdateNotificationReadStatus(ctx context.Context, db *sql.DB) (int64, error) {
	updateQuery := `UPDATE NOTIFICATION
				SET status = 1
				WHERE NOTIFICATION.status = 0
				AND  EXISTS (
					SELECT 1
					FROM READ_NOTIFICATIONS
					WHERE NOTIFICATION.TOUSER = READ_NOTIFICATIONS.OWNER
					AND NOTIFICATION.MESSAGETYPE = READ_NOTIFICATIONS.MESSAGETYPE
					AND NOTIFICATION.CREATEDAT < READ_NOTIFICATIONS.TIME
				    
				);`

	result, err := db.ExecContext(ctx, updateQuery)
	if err != nil {
		return 0, fmt.Errorf("Error updating NOTIFICATION records: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("Error getting the number of updated rows: %v", err)
	}

	return rowsAffected, nil
}

func truncateMessageContent(digest string) string {
	decodedDigest, err := url.QueryUnescape(digest)
	if err != nil {
		decodedDigest = digest // Use the original string if decoding fails
	}

	words := strings.Fields(decodedDigest)
	truncatedWords := []string{}
	charCount := 0

	for i, word := range words {
		// If the first word is longer than 20 characters, return the first 20 characters followed by '...'
		if i == 0 && len(word) > 20 {
			return word[:20] + "..."
			break
		}

		if i >= 4 || charCount+len(word) > 20 {
			break
		}

		truncatedWords = append(truncatedWords, word)
		charCount += len(word)
	}

	truncated := strings.Join(truncatedWords, " ")
	if len(truncated) < len(decodedDigest) {
		truncated += "..."
	}
	return truncated
}
