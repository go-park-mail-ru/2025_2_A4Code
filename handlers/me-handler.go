package handlers

import (
	ua "2025_2_a4code/internal/lib/user-actions"
	"encoding/json"
	"net/http"
)

func (h *Handlers) MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Неправильный метод", http.StatusMethodNotAllowed)
	}

	user, err := ua.GetCurrentUserData(r, SECRET, users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{
		"status": "200",
		"body":   user,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
