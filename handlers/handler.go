package handlers

import (
	md "2025_2_a4code/handlers/mock-data"
	ua "2025_2_a4code/internal/lib/user-actions"
	"2025_2_a4code/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var SECRET = []byte("secret")
var users md.MockDataSignup

type Handlers struct{}

func New() *Handlers {
	return &Handlers{}
}

func (handler *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
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

func (handler *Handlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	_, err := ua.CheckSession(r, SECRET)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
}

func (handler *Handlers) InboxHandler(w http.ResponseWriter, r *http.Request) {

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

func (handler *Handlers) SignupHandler(w http.ResponseWriter, r *http.Request) {

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
	for _, user := range users {
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
	users = append(users, newUser)

	// создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": credentials.Login,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	// подписываем
	session, err := token.SignedString(SECRET)

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
}

func (handler *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
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

func (handler *Handlers) MeHandler(w http.ResponseWriter, r *http.Request) {
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
