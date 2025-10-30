package inbox

import (
	"2025_2_a4code/internal/domain"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"log/slog"
	"time"

	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
)

var SECRET = []byte("secret") // TODO: убрать отсюда

type Sender struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type Message struct {
	Sender   Sender    `json:"sender"`
	Topic    string    `json:"topic"`
	Snippet  string    `json:"snippet"`
	Datetime time.Time `json:"datetime"`
	IsRead   bool      `json:"is_read"`
}
type Response struct {
	resp.Response
	Body domain.Messages `json:"body,omitempty"`
}

type HandlerInbox struct {
	profileUCase *profileUcase.ProfileUcase
	messageUCase *messageUcase.MessageUcase
	log          *slog.Logger
}

func New(profileUCase *profileUcase.ProfileUcase, messageUCase *messageUcase.MessageUcase, log *slog.Logger) *HandlerInbox {
	return &HandlerInbox{profileUCase: profileUCase, messageUCase: messageUCase, log: log}
}

func (h *HandlerInbox) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle /messages/inbox")

	if r.Method != http.MethodGet {
		log.Error(http.StatusText(http.StatusMethodNotAllowed))
		sendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		log.Error(err.Error())
		sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	messagesResponse := make([]Message, 0)
	messages, err := h.messageUCase.FindByProfileID(id)
	if err != nil {
		log.Error(err.Error())
		sendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for _, m := range messages {
		messagesResponse = append(messagesResponse, Message{
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
	}

	messagesInfo, err := h.messageUCase.GetMessagesInfo(id)
	if err != nil {
		log.Error(err.Error())
		sendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	messagesInfo.Messages = messagesResponse

	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(http.StatusOK),
			Message: "Письма получены",
		},
		Body: messagesInfo,
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error(err.Error())
		sendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {

	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(statusCode),
			Message: "Error: " + errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
