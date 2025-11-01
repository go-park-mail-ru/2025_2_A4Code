package inbox

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"log/slog"
	"time"

	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
)

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
}

type HandlerInbox struct {
	profileUCase *profileUcase.ProfileUcase
	messageUCase *messageUcase.MessageUcase
	log          *slog.Logger
	secret       []byte
}

func New(profileUCase *profileUcase.ProfileUcase, messageUCase *messageUcase.MessageUcase, log *slog.Logger, SECRET []byte) *HandlerInbox {
	return &HandlerInbox{
		profileUCase: profileUCase,
		messageUCase: messageUCase,
		log:          log,
		secret:       SECRET,
	}
}

func (h *HandlerInbox) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle /messages/inbox")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	messagesResponse := make([]Message, 0)
	messages, err := h.messageUCase.FindByProfileID(r.Context(), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
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

	messagesInfo, err := h.messageUCase.GetMessagesInfo(r.Context(), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	messagesInfo.Messages = messagesResponse

	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(http.StatusOK),
			Message: "success",
			Body:    messagesInfo,
		},
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
