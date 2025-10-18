package login

import (
	resp "2025_2_a4code/internal/lib/api/response"

	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Response struct {
	resp.Response
	Body struct {
		Message string `json:"message"`
	} `json:"body"`
}

type HandlerLogin struct {
	profileUCase *profileUcase.ProfileUcase
	JWTSecret    []byte
}

func New(ucP *profileUcase.ProfileUcase, secret []byte) *HandlerLogin {
	return &HandlerLogin{
		profileUCase: ucP,
		JWTSecret:    secret,
	}
}

func (h *HandlerLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Неправильный запрос", http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if req.Email == "" || req.Password == "" {
		sendErrorResponse(w, "Введите все поля формы", http.StatusBadRequest)
		return
	}

	// Преобразуем в UseCase запрос
	LoginReq := profileUcase.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// Вызываем usecase для входа
	userID, err := h.profileUCase.Login(r.Context(), LoginReq)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создаем JWT токен после успешной регистрации
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	session, err := token.SignedString(h.JWTSecret)
	if err != nil {
		sendErrorResponse(w, "Ошибка создания сессии", http.StatusInternalServerError)
		return
	}

	// Устанавливаем cookie
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    session,
		MaxAge:   3600,
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	// Отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Response: resp.Response{
			Status: "200",
		},
		Body: struct {
			Message string `json:"message"`
		}{
			Message: "Пользователь авторизован",
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		sendErrorResponse(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(statusCode),
			Error:  errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
