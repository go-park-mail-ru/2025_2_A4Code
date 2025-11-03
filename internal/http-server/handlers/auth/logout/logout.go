package logout

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"encoding/json"
	"log/slog"
	"net/http"
)

type Response struct {
	resp.Response
}

type HandlerLogout struct {
	log *slog.Logger
}

func New(log *slog.Logger) *HandlerLogout {
	return &HandlerLogout{
		log: log,
	}
}

func (h *HandlerLogout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle /auth/logout")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    struct{}{},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
