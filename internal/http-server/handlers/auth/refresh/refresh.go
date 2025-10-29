package refresh

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Response struct {
	resp.Response
	Body interface{} `json:"body,omitempty"`
}

type HandlerRefresh struct {
	JWTSecret []byte
}

func New(secret []byte) *HandlerRefresh {
	return &HandlerRefresh{
		JWTSecret: secret,
	}
}

func (h *HandlerRefresh) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем текущую сессию
	claims, err := session.CheckSession(r, h.JWTSecret)
	if err != nil {
		sendErrorResponse(w, "refresh token просрочен", http.StatusUnauthorized)
		return
	}

	// Извлекаем user_id из claims
	userID, ok := claims["user_id"].(float64)
	if !ok {
		sendErrorResponse(w, "Неверный токен", http.StatusUnauthorized)
		return
	}

	// Создаем новый JWT токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(userID),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	newSession, err := token.SignedString(h.JWTSecret)
	if err != nil {
		sendErrorResponse(w, "Ошибка создания сессии", http.StatusInternalServerError)
		return
	}

	// Устанавливаем новую cookie
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    newSession,
		MaxAge:   3600,
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	// Отправляем успешный ответ
	response := Response{
		Response: resp.Response{
			Status:  "200",
			Message: "Refresh token получен",
		},
		Body: struct{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		sendErrorResponse(w, "Внутренния ошибка сервера", http.StatusInternalServerError)
		return
	}
}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {
	response := Response{
		Response: resp.Response{
			Status:  strconv.Itoa(statusCode),
			Message: errorMsg,
		},
		Body: struct{}{},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
