package inbox

//go:generate mockgen -source=$GOFILE -destination=./mocks/mock_inbox_usecase.go -package=mocks

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type MessageUsecase interface {
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetMessagesInfoWithPagination(ctx context.Context, profileID int64) (domain.Messages, error)
}

type AvatarUsecase interface {
	GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error)
}

type Sender struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type Message struct {
	ID       string    `json:"id"`
	Sender   Sender    `json:"sender"`
	Topic    string    `json:"topic"`
	Snippet  string    `json:"snippet"`
	Datetime time.Time `json:"datetime"`
	IsRead   bool      `json:"is_read"`
}

type PaginationInfo struct {
	HasNext           bool   `json:"has_next"`
	NextLastMessageID int64  `json:"next_last_message_id,omitempty"`
	NextLastDatetime  string `json:"next_last_datetime,omitempty"`
}

type InboxResponse struct {
	MessageTotal  int            `json:"message_total"`
	MessageUnread int            `json:"message_unread"`
	Messages      []Message      `json:"messages"`
	Pagination    PaginationInfo `json:"pagination"`
}

type Response struct {
	resp.Response
}

type HandlerInbox struct {
	messageUCase MessageUsecase
	avatarUCase  AvatarUsecase
	secret       []byte
}

func New(messageUCase MessageUsecase, avatarUCase AvatarUsecase, SECRET []byte) *HandlerInbox {
	return &HandlerInbox{
		messageUCase: messageUCase,
		avatarUCase:  avatarUCase,
		secret:       SECRET,
	}
}

func (h *HandlerInbox) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("handle /messages/inbox")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	lastMessageIDStr := r.URL.Query().Get("last_message_id")
	lastDatetimeStr := r.URL.Query().Get("last_datetime")
	limitStr := r.URL.Query().Get("limit")

	var lastMessageID int64
	var lastDatetime time.Time

	if lastMessageIDStr != "" {
		if id, err := strconv.ParseInt(lastMessageIDStr, 10, 64); err == nil {
			lastMessageID = id
		}
	}

	if lastDatetimeStr != "" {
		if dt, err := time.Parse(time.RFC3339, lastDatetimeStr); err == nil {
			lastDatetime = dt
		}
	}

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	messages, err := h.messageUCase.FindByProfileIDWithKeysetPagination(r.Context(), id, lastMessageID, lastDatetime, limit)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	messagesInfo, err := h.messageUCase.GetMessagesInfoWithPagination(r.Context(), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	messagesResponse := make([]Message, 0, len(messages))
	var nextLastMessageID int64
	var nextLastDatetime time.Time

	for _, m := range messages {
		messageID, _ := strconv.ParseInt(m.ID, 10, 64)
		if err := h.enrichSenderAvatar(r.Context(), &m.Sender); err != nil {
			log.Warn("failed to enrich sender avatar: " + err.Error())
		}

		messagesResponse = append(messagesResponse, Message{
			ID: m.ID,
			Sender: Sender{
				Email:    m.Email,
				Username: m.Username,
				Avatar:   m.Avatar,
			},
			Topic:    m.Topic,
			Snippet:  m.Snippet,
			Datetime: m.Datetime,
			IsRead:   m.IsRead,
		})

		nextLastMessageID = messageID
		nextLastDatetime = m.Datetime
	}

	inboxResponse := InboxResponse{
		MessageTotal:  messagesInfo.MessageTotal,
		MessageUnread: messagesInfo.MessageUnread,
		Messages:      messagesResponse,
		Pagination: PaginationInfo{
			HasNext:           len(messages) == limit, // если получили полную страницу, значит есть еще
			NextLastMessageID: nextLastMessageID,
			NextLastDatetime:  nextLastDatetime.Format(time.RFC3339),
		},
	}

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    inboxResponse,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}

func (h *HandlerInbox) enrichSenderAvatar(ctx context.Context, sender *domain.Sender) error {
	if sender == nil || sender.Avatar == "" {
		return nil
	}

	objectName := sender.Avatar
	if strings.HasPrefix(objectName, "http://") || strings.HasPrefix(objectName, "https://") {
		parsed, err := url.Parse(objectName)
		if err != nil {
			return err
		}
		objectName = strings.TrimPrefix(parsed.Path, "/")
	}

	objectName = strings.TrimLeft(objectName, "/")
	if objectName == "" {
		return nil
	}

	if idx := strings.Index(objectName, "/"); idx != -1 {
		prefix := objectName[:idx]
		if strings.EqualFold(prefix, "avatars") {
			objectName = objectName[idx+1:]
		}
	}

	if objectName == "" {
		return nil
	}

	url, err := h.avatarUCase.GetAvatarPresignedURL(ctx, objectName, 15*time.Minute)
	if err != nil {
		return err
	}

	sender.Avatar = url.String()
	return nil
}
