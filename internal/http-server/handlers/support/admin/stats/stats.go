package adminstats

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type AppealUsecase interface {
	GetAppealsStats(ctx context.Context) (domain.SupportStats, error)
}

type ProfileUsecase interface {
	FindInfoByID(ctx context.Context, id int64) (domain.ProfileInfo, error)
}

type Handler struct {
	appealUCase  AppealUsecase
	profileUCase ProfileUsecase
	secret       []byte
}

type StatsResponse struct {
	Total      int `json:"total_appeals"`
	Open       int `json:"open_appeals"`
	InProgress int `json:"in_progress_appeals"`
	Closed     int `json:"closed_appeals"`
}

func New(appealUCase AppealUsecase, profileUCase ProfileUsecase, secret []byte) *Handler {
	return &Handler{appealUCase: appealUCase, profileUCase: profileUCase, secret: secret}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	profileID, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if ok := h.ensureAdmin(r.Context(), profileID); !ok {
		resp.SendErrorResponse(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	stats, err := h.appealUCase.GetAppealsStats(r.Context())
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := StatsResponse{
		Total:      stats.TotalAppeals,
		Open:       stats.OpenAppeals,
		InProgress: stats.InProgressAppeals,
		Closed:     stats.ClosedAppeals,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Response{Status: http.StatusOK, Message: "success", Body: response})
}

func (h *Handler) ensureAdmin(ctx context.Context, profileID int64) bool {
	info, err := h.profileUCase.FindInfoByID(ctx, profileID)
	if err != nil {
		return false
	}

	role := strings.ToLower(strings.TrimSpace(info.Role))
	return role == "admin" || role == "support"
}
