package message_page

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	avatar "2025_2_a4code/internal/usecase/avatar"
	"2025_2_a4code/internal/usecase/message"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Sender struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type File struct {
	Name        string `json:"name"`
	FileType    string `json:"file_type"`
	Size        int64  `json:"size"`
	StoragePath string `json:"storage_path"`
}

type Files []File

type Message struct {
	Topic    string    `json:"topic"`
	Text     string    `json:"text"`
	Datetime time.Time `json:"datetime"`
	ThreadId string    `json:"thread_id"`
	Sender
	Files
}
type Response struct {
	resp.Response
}

type HandlerMessagePage struct {
	profileUCase *profile.ProfileUcase
	messageUCase *message.MessageUcase
	avatarUCase  *avatar.AvatarUcase
	secret       []byte
}

func New(profileUCase *profile.ProfileUcase, messageUCase *message.MessageUcase, avatarUCase *avatar.AvatarUcase, SECRET []byte) *HandlerMessagePage {
	return &HandlerMessagePage{
		profileUCase: profileUCase,
		messageUCase: messageUCase,
		avatarUCase:  avatarUCase,
		secret:       SECRET,
	}
}

func (h *HandlerMessagePage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Info("handle messages/{message_id}")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	path := r.URL.Path
	messageIDStr := strings.TrimPrefix(path, "/messages/")
	messageIDStr = strings.TrimSuffix(messageIDStr, "/")

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	fullMessage, err := h.messageUCase.FindFullByMessageID(r.Context(), int64(messageID), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.messageUCase.MarkMessageAsRead(r.Context(), int64(messageID), id); err != nil {
		log.Warn("failed to mark message as read: " + err.Error())
	}

	if err := h.enrichSenderAvatar(r.Context(), &fullMessage.Sender); err != nil {
		log.Warn("failed to enrich sender avatar: " + err.Error())
	}

	filesResponse := make([]File, len(fullMessage.Files))
	for i, file := range fullMessage.Files {
		filesResponse[i] = File{
			Name:     file.Name,
			FileType: file.FileType,
			Size:     file.Size,
		}
	}

	messageResponse := Message{
		Topic:    fullMessage.Topic,
		Text:     fullMessage.Text,
		Datetime: fullMessage.Datetime,
		Sender: Sender{
			Email:    fullMessage.Email,
			Username: fullMessage.Username,
			Avatar:   fullMessage.Avatar,
		},
		ThreadId: fullMessage.ThreadRoot,
		Files:    filesResponse,
	}

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    messageResponse,
		},
	}

	err = json.NewEncoder(w).Encode(response)
}

func (h *HandlerMessagePage) enrichSenderAvatar(ctx context.Context, sender *domain.Sender) error {
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
