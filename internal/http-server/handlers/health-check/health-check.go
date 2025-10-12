package health_check

import (
	"2025_2_a4code/internal/http-server/handlers"
	ua "2025_2_a4code/internal/lib/user-actions"
	"net/http"
)

func (h *handlers.Handlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	_, err := ua.CheckSession(r, handlers.SECRET)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
}
