package message_page

import (
	"2025_2_a4code/internal/domain"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/message"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
	"strconv"
)

var SECRET = []byte("secret") // TODO: убрать отсюда

type Response struct {
	resp.Response
	Body domain.FullMessage `json:"body"`
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
	}

	_, err := session.GetProfileID(r, SECRET)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
	}

	messageID, err := strconv.Atoi(r.URL.Query().Get("message_id"))
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
	}
	fullMessage, err := h.messageUCase.FindFullByMessageID(int64(messageID))
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
	}

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(http.StatusOK),
		},
		Body: fullMessage,
	}

	err = json.NewEncoder(w).Encode(response)
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
