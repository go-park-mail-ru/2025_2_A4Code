package inbox

import (
	"2025_2_a4code/internal/domain"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"

	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
)

var SECRET = []byte("secret") // TODO: убрать отсюда

type Response struct {
	resp.Response
	Body domain.Messages `json:"body,omitempty"`
}

type HandlerInbox struct {
	profileUCase *profileUcase.ProfileUcase
	messageUCase *messageUcase.MessageUcase
}

func New(ucP *profileUcase.ProfileUcase, ucM *messageUcase.MessageUcase) *HandlerInbox {
	return &HandlerInbox{profileUCase: ucP, messageUCase: ucM}
}

func (h *HandlerInbox) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	messagesResponse, err := h.messageUCase.GetMessagesInfo(id)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
	}

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(http.StatusOK),
		},
		Body: messagesResponse,
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		sendErrorResponse(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(statusCode),
			Error:  errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
