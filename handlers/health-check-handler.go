package handlers

import (
	ua "2025_2_a4code/internal/lib/user-actions"
	"net/http"
)

func (h *Handlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	_, err := ua.CheckSession(r, SECRET)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
}
