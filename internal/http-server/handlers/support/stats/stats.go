package stats

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"net/http"
	"time"
)

type StatsUsecase interface {
	FindAppealByProfileID(ctx context.Context, profileID int64) (domain.Appeal, error)
	FindAppealsInfoByProfileID(ctx context.Context, profileID int64) (domain.AppealsInfo, error)
	GetUserRoleByProfileID(ctx context.Context, profileID int64) (string, error)
}

type Response struct {
	resp.Response
}
type Appeal struct {
	Topic     string    `json:"topic"`
	Text      string    `json:"text"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AppealsResponse struct {
	TotalAppeals      int    `json:"total_appeals"`
	OpenAppeals       int    `json:"open_appeals"`
	InProgressAppeals int    `json:"in_progress_appeals"`
	ClosedAppeals     int    `json:"closed_appeals"`
	LastAppeal        Appeal `json:"last_appeal"`
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
	log.Debug("Handle support/stats")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	lastAppeal, err := h.appealsUsecase.FindAppealByProfileID(r.Context(), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	appealsInfo, err := h.appealsUsecase.FindAppealsInfoByProfileID(r.Context(), id)
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
		LastAppeal: Appeal{
			Topic:     lastAppeal.Topic,
			Text:      lastAppeal.Text,
			Status:    lastAppeal.Status,
			CreatedAt: lastAppeal.CreatedAt,
			UpdatedAt: lastAppeal.UpdatedAt,
		},
	}
}
