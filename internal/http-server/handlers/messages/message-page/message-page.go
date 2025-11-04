package message_page

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/message"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
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
	secret       []byte
}

func New(profileUCase *profile.ProfileUcase, messageUCase *message.MessageUcase, SECRET []byte) *HandlerMessagePage {
	return &HandlerMessagePage{
		profileUCase: profileUCase,
		messageUCase: messageUCase,
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
