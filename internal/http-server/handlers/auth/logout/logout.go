package logout

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"encoding/json"
	"net/http"
)

type Response struct {
	resp.Response
}

type HandlerLogout struct {
}

func New() *HandlerLogout {
	return &HandlerLogout{}
}

func (h *HandlerLogout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(http.StatusOK),
			Message: "Успешный выход из почты",
			Body:    struct{}{},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		resp.SendErrorResponse(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}
