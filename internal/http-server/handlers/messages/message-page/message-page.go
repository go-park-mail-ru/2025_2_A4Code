package message_page

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/message"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

var SECRET = []byte("secret") // TODO: убрать отсюда

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
	Body Message `json:"body"`
}

type HandlerMessagePage struct {
	profileUCase *profile.ProfileUcase
	messageUCase *message.MessageUcase
}

func New(ucP *profile.ProfileUcase, usM *message.MessageUcase) *HandlerMessagePage {
	return &HandlerMessagePage{profileUCase: ucP, messageUCase: usM}
}

func (h *HandlerMessagePage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	messageID, err := strconv.Atoi(r.URL.Query().Get("message_id"))
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fullMessage, err := h.messageUCase.FindFullByMessageID(int64(messageID), id)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
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
			Status:  http.StatusText(http.StatusOK),
			Message: "Письмо отправлено",
		},
		Body: messageResponse,
	}

	err = json.NewEncoder(w).Encode(response)
}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {

	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(statusCode),
			Message: "Ошибка: " + errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
