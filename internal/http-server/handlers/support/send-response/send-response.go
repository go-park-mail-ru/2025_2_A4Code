package send_response

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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

	_, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 2 {
		resp.SendErrorResponse(w, "missing appeal ID", http.StatusBadRequest)
		return
	}

	appealIDStr := parts[len(parts)-1]

	appealID, err := strconv.Atoi(appealIDStr)
	if err != nil {
		log.Error("Failed to parse appeal ID: " + err.Error()) // Improved logging
		resp.SendErrorResponse(w, "invalid appeal ID format", http.StatusBadRequest)
		return
	}

	var req UpdateAppealRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp.SendErrorResponse(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.appealUCase.UpdateAppeal(r.Context(), int64(appealID), req.Text, req.Status); err != nil {
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
