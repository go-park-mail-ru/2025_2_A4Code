package handlers

import (
	md "2025_2_a4code/handlers/mock-data"
	ua "2025_2_a4code/internal/lib/user-actions"
	"encoding/json"
	"net/http"
)

func (h *Handlers) InboxHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Выдача списка писем конкретного пользователя
	_, err := ua.GetCurrentUserData(r, SECRET, users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	// Тестовые данные в виде map[string]interface{}
	res := md.New()

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

}
