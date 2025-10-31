package login

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"log/slog"
	"strings"

	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Request struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Response struct {
	resp.Response
}

type HandlerLogin struct {
	profileUCase *profileUcase.ProfileUcase
	log          *slog.Logger
	JWTSecret    []byte
}

func New(ucP *profileUcase.ProfileUcase, log *slog.Logger, secret []byte) *HandlerLogin {
	return &HandlerLogin{
		profileUCase: ucP,
		log:          log,
		JWTSecret:    secret,
	}
}

func (h *HandlerLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle /auth/login")

	if r.Method != http.MethodPost {
		log.Error(http.StatusText(http.StatusMethodNotAllowed))
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(http.StatusText(http.StatusBadRequest))
		resp.SendErrorResponse(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if req.Login == "" || req.Password == "" {
		resp.SendErrorResponse(w, "Введите все поля формы", http.StatusBadRequest)
		return
	}

	username := req.Login
	if strings.Contains(req.Login, "@") {
		parts := strings.Split(req.Login, "@")
		if len(parts) > 0 && parts[0] != "" {
			username = parts[0]
		} else {
			resp.SendErrorResponse(w, "Неправильный формат логина или почты", http.StatusBadRequest)
			return
		}
	}

	// Преобразуем в UseCase запрос
	LoginReq := profileUcase.LoginRequest{
		Username: username,
		Password: req.Password,
	}

	// Вызываем usecase для входа
	userID, err := h.profileUCase.Login(r.Context(), LoginReq)
	if err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создаем JWT токен после успешной регистрации
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	session, err := token.SignedString(h.JWTSecret)
	if err != nil {
		resp.SendErrorResponse(w, "Ошибка создания сессии", http.StatusInternalServerError)
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
			Status:  "200",
			Message: "Вы успешно авторизованы",
			Body:    struct{}{},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		resp.SendErrorResponse(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}
