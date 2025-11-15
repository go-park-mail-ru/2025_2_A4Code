package send_appeal

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
)

type AppealUsecase interface {
	SaveAppeal(ctx context.Context, profileID int64, topic, text string) error
}

type Request struct {
	Topic string `json:"topic"`
	Text  string `json:"text"`
}

type Response struct {
	resp.Response
}

type HandlerSendAppeal struct {
	appealUCase AppealUsecase
	secret      []byte
}

func New(appealUCase AppealUsecase, SECRET []byte) *HandlerSendAppeal {
	return &HandlerSendAppeal{
		appealUCase: appealUCase,
		secret:      SECRET,
	}
}

func (h *HandlerSendAppeal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("handle support/send-appeal")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		resp.SendErrorResponse(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	err = h.appealUCase.SaveAppeal(r.Context(), id, req.Topic, req.Text)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    struct{}{},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
