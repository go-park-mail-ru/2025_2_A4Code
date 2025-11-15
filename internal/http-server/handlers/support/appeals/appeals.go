package appeals

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"net/http"
	"strconv"
	"time"
)

type AppealUsecase interface {
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID, lastAppealID int64, lastDatetime time.Time, limit int) ([]domain.Appeal, error)
}

type Appeal struct {
	Topic     string    `json:"topic"`
	Text      string    `json:"text"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AppealsInfo struct {
	Appeals    []Appeal       `json:"appeals"`
	Pagination PaginationInfo `json:"pagination"`
}

type PaginationInfo struct {
	HasNext           bool   `json:"has_next"`
	NextLastMessageID int64  `json:"next_last_message_id,omitempty"`
	NextLastDatetime  string `json:"next_last_datetime,omitempty"`
}

type Response struct {
	resp.Response
}

type HandlerAppeals struct {
	appealUCase AppealUsecase
	secret      []byte
}

func New(appealUCase AppealUsecase, SECRET []byte) *HandlerAppeals {
	return &HandlerAppeals{
		appealUCase: appealUCase,
		secret:      SECRET,
	}
}

func (h *HandlerAppeals) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("Handle support/appeals")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	lastAppealIDStr := r.URL.Query().Get("last_message_id")
	lastDatetimeStr := r.URL.Query().Get("last_datetime")
	limitStr := r.URL.Query().Get("limit")

	var lastAppealID int64
	var lastDatetime time.Time

	if lastAppealIDStr != "" {
		if id, err := strconv.ParseInt(lastAppealIDStr, 10, 64); err == nil {
			lastAppealID = id
		}
	}

	if lastDatetimeStr != "" {
		if dt, err := time.Parse(time.RFC3339, lastDatetimeStr); err == nil {
			lastDatetime = dt
		}
	}

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	appeals, err := h.appealUCase.FindByProfileIDWithKeysetPagination(r.Context(), id, lastAppealID, lastDatetime, limit)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	appealsResponse := make([]Appeal, len(appeals))
	for _, appeal := range appeals {
		appealsResponse = append(appealsResponse, Appeal{
			Topic:     appeal.Topic,
			Text:      appeal.Text,
			Status:    appeal.Status,
			CreatedAt: appeal.CreatedAt,
			UpdatedAt: appeal.UpdatedAt,
		})
	}

}
