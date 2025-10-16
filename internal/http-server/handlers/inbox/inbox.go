package inbox

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/lib/session"

	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
)

// TODO: убрать отсюда
var SECRET = []byte("secret")

type SuccessResponse struct {
	Status string          `json:"Status"`
	Body   domain.Messages `json:"body"`
}

type ErrorResponse struct {
	Status string `json:"Status"`
	Body   struct {
		Error string `json:"error"`
	} `json:"Body"`
}

type InboxHandler struct {
	profileUCase *profileUcase.ProfileUcase
	messageUCase *messageUcase.MessageUcase
}

func New(ucBP *profileUcase.ProfileUcase, ucM *messageUcase.MessageUcase) *InboxHandler {
	return &InboxHandler{profileUCase: ucBP, messageUCase: ucM}
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

	messagesResponse, err := h.messageUCase.GetMessagesInfo(id)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
	}

	resp := SuccessResponse{
		Status: "200",
		Body:   messagesResponse,
	}

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
