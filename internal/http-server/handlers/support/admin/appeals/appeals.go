package adminappeals

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AppealUsecase interface {
	FindAllAppeals(ctx context.Context, lastAppealID int64, lastDatetime time.Time, limit int) ([]domain.AdminAppeal, error)
}

type ProfileUsecase interface {
	FindInfoByID(ctx context.Context, id int64) (domain.ProfileInfo, error)
}

type Handler struct {
	appealUCase  AppealUsecase
	profileUCase ProfileUsecase
	secret       []byte
}

type AppealsResponse struct {
	Appeals []AdminAppealResponse `json:"appeals"`
}

type AdminAppealResponse struct {
	ID          int64     `json:"id"`
	Topic       string    `json:"topic"`
	Text        string    `json:"text"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
}

func New(appealUCase AppealUsecase, profileUCase ProfileUsecase, secret []byte) *Handler {
	return &Handler{
		appealUCase:  appealUCase,
		profileUCase: profileUCase,
		secret:       secret,
	}
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

	params := r.URL.Query()
	lastID, _ := parseInt64(params.Get("last_id"))
	lastDatetime := parseTime(params.Get("last_datetime"))
	limit := parseLimit(params.Get("limit"), 50)

	appeals, err := h.appealUCase.FindAllAppeals(r.Context(), lastID, lastDatetime, limit)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := AppealsResponse{Appeals: make([]AdminAppealResponse, 0, len(appeals))}
	for _, appeal := range appeals {
		response.Appeals = append(response.Appeals, AdminAppealResponse{
			ID:          appeal.ID,
			Topic:       appeal.Topic,
			Text:        appeal.Text,
			Status:      appeal.Status,
			CreatedAt:   appeal.CreatedAt,
			UpdatedAt:   appeal.UpdatedAt,
			AuthorEmail: appeal.AuthorEmail,
			AuthorName:  appeal.AuthorName,
		})
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

func parseInt64(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 200 {
		return parsed
	}
	return fallback
}
