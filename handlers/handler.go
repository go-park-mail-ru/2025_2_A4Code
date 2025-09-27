package handlers

import (
	test_data "2025_2_a4code/handlers/test-data"
	"2025_2_a4code/models"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var SECRET = []byte("secret")

type Handlers struct{}

func (handler *Handlers) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

}

func New() *Handlers {
	return &Handlers{}
}

func (handler *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var credentials models.BaseUser
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// проверка логина и пароля
	if credentials.Login != "admin" || credentials.Password != "admin" {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	// создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": credentials.Login,
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

func (handler *Handlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")

	if err != nil {
		http.Error(w, "Сессия не найдена", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		// проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return SECRET, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Неверный токен", http.StatusUnauthorized)
		return
	}

	// Извлекаем claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Ошибка при чтении данных токена", http.StatusInternalServerError)
		return
	}

	// извлекаем логин
	login, ok := claims["login"].(string)
	if !ok {
		http.Error(w, "Логин не найден в токене", http.StatusBadRequest)
		return
	}

	if exp, ok := claims["exp"].(float64); !ok {
		if time.Now().Unix() > int64(exp) {
			http.Error(w, "Токен просрочен", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{
		"status": "200",
		"body": struct {
			Message string `json:"message"`
		}{"Hello, " + login},
	})
	if err != nil {
		// ошибка
		return
	}
}

func (handler *Handlers) MainPageHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	res := test_data.New()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(&res)

	if err != nil {
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

}
