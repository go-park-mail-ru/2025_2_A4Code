package appeals

import (
	"2025_2_a4code/internal/domain"
	resp "2025_2_a4code/internal/lib/api/response"
	"context"
	"net/http"
	"time"
)

type AppealUsecase interface {
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID, lastAppealID int64, lastDatetime time.Time, limit int) ([]domain.Appeal, error)
}

type Appeal struct {
	Topic     string `json:"topic"`
	Text      string `json:"text"`
	status    string `json:"status"`
	createdAt string `json:"created_at"`
	updatedAt string `json:"updated_at"`
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

}
