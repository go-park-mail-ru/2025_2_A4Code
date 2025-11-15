package send_response

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/appeal"
	"context"
	"encoding/json"
	"net/http"
)

type AppealUsecase interface {
	UpdateAppeal(ctx context.Context, appealID int64, text string, status string) error
}

type Response struct {
	resp.Response
}

type UpdateAppealRequest struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

type HandlerSendAppealResponse struct {
	appealUCase AppealUsecase
	secret      []byte
}

func New(appealUCase AppealUsecase, secret []byte) *HandlerSendAppealResponse {
	return &HandlerSendAppealResponse{
		appealUCase: appealUCase,
		secret:      secret,
	}
}

func (h *HandlerSendAppealResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Info("handle user/profile (PUT)")

	if r.Method != http.MethodPut {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req UpdateAppealRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp.SendErrorResponse(w, "invalid request body", http.StatusBadRequest)
		return
	}

	updateReq := appeal.UpdateAppealRequest{
		Text:   req.Text,
		Status: req.Status,
	}

	if err := h.appealUCase.UpdateAppeal(r.Context(), id, req.Text, req.Status); err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Response{
		Status:  http.StatusOK,
		Message: "success",
	})
}
