package supstats

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
)

type StatsUsecase interface {
	FindAllAppealsStats(ctx context.Context) (domain.AppealsInfo, error)
}

type Response struct {
	resp.Response
}

type AppealsResponse struct {
	TotalAppeals      int `json:"total_appeals"`
	OpenAppeals       int `json:"open_appeals"`
	InProgressAppeals int `json:"in_progress_appeals"`
	ClosedAppeals     int `json:"closed_appeals"`
}

type HandlerAppeal struct {
	appealsUsecase StatsUsecase
	secret         []byte
}

func New(appealsUsecase StatsUsecase, SECRET []byte) *HandlerAppeal {
	return &HandlerAppeal{
		appealsUsecase: appealsUsecase,
		secret:         SECRET,
	}
}

func (h *HandlerAppeal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("Handle support/userstats")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	appealsInfo, err := h.appealsUsecase.FindAllAppealsStats(r.Context())
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := AppealsResponse{
		TotalAppeals:      appealsInfo.TotalAppeals,
		OpenAppeals:       appealsInfo.OpenAppeals,
		InProgressAppeals: appealsInfo.InProgressAppeals,
		ClosedAppeals:     appealsInfo.ClosedAppeals,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
