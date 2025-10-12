package handlers

import (
	"2025_2_a4code/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	var credentials models.BaseUser
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// проверка логина и пароля
	found := false
	for _, user := range users {
		if credentials.Login == user["login"] && credentials.Password == user["password"] {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	// создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": credentials.Login,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})

	// подписываем
	session, err := token.SignedString(SECRET)

	if err != nil {
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    session,
		MaxAge:   3600,
		HttpOnly: true,
		Path:     "/",
	}

	// ставим куки
	http.SetCookie(w, cookie)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{
		"status": "200",
		"body": struct {
			Message string `json:"message"`
		}{"Пользователь авторизован"},
	})

	if err != nil {
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}
