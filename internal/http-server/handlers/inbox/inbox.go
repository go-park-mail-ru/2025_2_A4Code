package inbox

import (
	"2025_2_a4code/internal/lib/session"
	message_ucase "2025_2_a4code/internal/usecase/message-ucase"
	"2025_2_a4code/internal/usecase/profile-ucase"
	profilemessage_ucase "2025_2_a4code/internal/usecase/profilemessage-ucase"
	"encoding/json"
	"net/http"
	"strconv"
)

// TODO: убрать отсюда
var SECRET = []byte("secret")

type SuccessResponse struct {
	Status string `json:"Status"`
	Body   struct {
		MessageTotal  int       `json:"message_total"`
		MessageUnread int       `json:"message_unread"`
		Messages      []Message `json:"messages"`
	} `json:"body"`
}

type ErrorResponse struct {
	Status string `json:"Status"`
	Body   struct {
		Error string `json:"error"`
	} `json:"Body"`
}

type Message struct {
	ID       string `json:"id"`
	Sender   Sender `json:"sender"`
	Topic    string `json:"topic"`
	Snippet  string `json:"snippet"`
	Datetime string `json:"datetime"`
	IsRead   bool   `json:"is_read"`
	Folder   string `json:"folder"`
}

type Sender struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}
type InboxHandler struct {
	BaseProfileUCase    *profile_ucase.BaseProfileUcase
	profileMessageUCase *profilemessage_ucase.ProfileMessageUcase
	messageUCase        *message_ucase.MessageUcase
}

func New(ucBP *profile_ucase.BaseProfileUcase, ucPM *profilemessage_ucase.ProfileMessageUcase, ucM *message_ucase.MessageUcase) *InboxHandler {
	return &InboxHandler{BaseProfileUCase: ucBP, profileMessageUCase: ucPM, messageUCase: ucM}
}

func (h *InboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	messages, err := h.profileMessageUCase.FindByProfileID(id)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messagesResponse := []Message{}

	unread := len(messages)
	for _, message := range messages {
		if message.ReadStatus {
			unread--
		}
		messageInfo, err := h.messageUCase.FindByID(message.MessageID)
		if err != nil {
			sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		}

		sender, err := h.BaseProfileUCase.FindByID(messageInfo.SenderBaseProfileID)
		if err != nil {
			sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		}

		messagesResponse = append(messagesResponse, Message{
			ID: strconv.FormatInt(messageInfo.ID, 10),
			Sender: Sender{
				Email:    sender.Username + sender.Domain,
				Username: sender.Username,
				Avatar:   "",
			},
			Topic:    messageInfo.Topic,
			Snippet:  messageInfo.Text[:10] + "...",
			Datetime: messageInfo.DateOfDispatch.String(),
			IsRead:   message.ReadStatus,
			Folder:   "",
		})
	}

	resp := SuccessResponse{
		Status: "200",
	}
	resp.Body.MessageTotal = len(messagesResponse)
	resp.Body.MessageUnread = unread
	resp.Body.Messages = messagesResponse

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&resp)
	if err != nil {
		sendErrorResponse(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {
	response := ErrorResponse{
		Status: http.StatusText(statusCode),
	}
	response.Body.Error = errorMsg

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
