package inbox

import (
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase"
	"encoding/json"
	"net/http"
)

// TODO: убрать отсюда
var SECRET = []byte("secret")

type InboxHandler struct {
	profileUcase *usecase.ProfileUcase
}

func New(uc *usecase.ProfileUcase) *InboxHandler {
	return &InboxHandler{profileUcase: uc}
}

func (h *InboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	profile, err := h.profileUcase.GetByID(id)
	if err != nil {

	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&profile)
	if err != nil {
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

}
