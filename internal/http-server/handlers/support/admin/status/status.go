package adminstatus

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
)

type AppealUsecase interface {
	UpdateAppealStatus(ctx context.Context, appealID int64, status string) error
}

type ProfileUsecase interface {
	FindInfoByID(ctx context.Context, id int64) (domain.ProfileInfo, error)
}

type Handler struct {
	appealUCase  AppealUsecase
	profileUCase ProfileUsecase
	secret       []byte
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

func New(appealUCase AppealUsecase, profileUCase ProfileUsecase, secret []byte) *Handler {
	return &Handler{appealUCase: appealUCase, profileUCase: profileUCase, secret: secret}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	if r.Method != http.MethodPatch {
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

	appealID, err := extractAppealID(r.URL.Path)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	status, ok := normalizeStatus(req.Status)
	if !ok {
		resp.SendErrorResponse(w, "invalid status", http.StatusBadRequest)
		return
	}

	if err := h.appealUCase.UpdateAppealStatus(r.Context(), appealID, status); err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Response{Status: http.StatusOK, Message: "success", Body: struct{}{}})
}

func extractAppealID(path string) (int64, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}

func normalizeStatus(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "open":
		return "open", true
	case "closed":
		return "closed", true
	case "in progress", "in_progress":
		return "in progress", true
	default:
		return "", false
	}
}

func (h *Handler) ensureAdmin(ctx context.Context, profileID int64) bool {
	info, err := h.profileUCase.FindInfoByID(ctx, profileID)
	if err != nil {
		return false
	}

	role := strings.ToLower(strings.TrimSpace(info.Role))
	return role == "admin" || role == "support"
}
