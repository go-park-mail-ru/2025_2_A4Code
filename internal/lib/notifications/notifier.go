package notifications

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
)

type NotificationType string

const (
	TypeNewMessage     NotificationType = "NEW_MESSAGE"
	TypeMessageRead    NotificationType = "MESSAGE_READ"
	TypeMessageSent    NotificationType = "MESSAGE_SENT"
	TypeMessageSpam    NotificationType = "MESSAGE_SPAM"
	TypeMessageDeleted NotificationType = "MESSAGE_DELETED"
)

type Notification struct {
	Type       NotificationType `json:"type"`
	UserID     int64            `json:"user_id"`
	MessageID  int64            `json:"message_id"`
	FromEmail  string           `json:"from,omitempty"`
	FromID     int64            `json:"from_id,omitempty"`
	Subject    string           `json:"subject,omitempty"`
	Preview    string           `json:"preview,omitempty"`
	Timestamp  int64            `json:"timestamp"`
	IsInternal bool             `json:"is_internal"`
}

type Notifier struct {
	redis *redis.Client
}

func NewNotifier(redis *redis.Client) *Notifier {
	return &Notifier{
		redis: redis,
	}
}

func (n *Notifier) IsInternalEmail(email string) bool {
	if n == nil {
		return false
	}
	return strings.HasSuffix(email, "@flintmail.ru")
}

func (n *Notifier) NotifyNewMessage(
	ctx context.Context,
	recipientID int64,
	messageID int64,
	fromEmail string,
	fromID int64,
	subject string,
	preview string,
) error {
	if n.redis == nil {
		return nil
	}

	isInternal := n.IsInternalEmail(fromEmail)

	notification := Notification{
		Type:       TypeNewMessage,
		UserID:     recipientID,
		MessageID:  messageID,
		FromEmail:  fromEmail,
		FromID:     fromID,
		Subject:    subject,
		Preview:    preview,
		Timestamp:  time.Now().Unix(),
		IsInternal: isInternal,
	}

	return n.publish(ctx, notification)
}

func (n *Notifier) publish(ctx context.Context, notification Notification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	slog.Debug("Publishing notification",
		"type", notification.Type,
		"user_id", notification.UserID,
		"message_id", notification.MessageID)

	return n.redis.Publish(ctx, "notifications", data).Err()
}
