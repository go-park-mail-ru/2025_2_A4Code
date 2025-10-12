package logout

import (
	handlers2 "2025_2_a4code/internal/http-server/handlers"
	"encoding/json"
	"net/http"
)

func (h *handlers2.Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "200",
		"body": struct {
			Message string `json:"message"`
		}{"Logged out"},
	})
}
