package signup

import (
	handlers2 "2025_2_a4code/internal/http-server/handlers"
	"2025_2_a4code/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (h *handlers2.Handlers) SignupHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	var credentials models.RegisteredUser
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// проверка логина
	for _, user := range handlers2.users {
		if credentials.Login == user["login"] {
			http.Error(w, "Пользователь с таким логином уже существует", http.StatusUnauthorized)
			return
		}
	}

	newUser := map[string]string{
		"login":       credentials.Login,
		"password":    credentials.Password,
		"username":    credentials.Username,
		"dateofbirth": credentials.DateOfBirth,
		"gender":      credentials.Gender,
	}

	//записываем в мап
	handlers2.users = append(handlers2.users, newUser)

	// создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": credentials.Login,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	// подписываем
	session, err := token.SignedString(handlers2.SECRET)

	if err != nil {
		http.Error(w, "Ошибка регистрации", http.StatusInternalServerError)
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
		}{"Пользователь зарегистрирован"},
	})

	if err != nil {
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}
