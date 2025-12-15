package login

//go:generate mockgen -source=$GOFILE -destination=./mocks/mock_profile_usecase.go -package=mocks

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	valid "2025_2_a4code/internal/lib/validation"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"

	"2025_2_a4code/internal/usecase/profile"

	"github.com/golang-jwt/jwt/v5"
)

type Request struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ProfileUsecase interface {
	Login(ctx context.Context, req profile.LoginRequest) (int64, error)
}

type Response struct {
	resp.Response
}

type HandlerLogin struct {
	profileUCase ProfileUsecase
	JWTSecret    []byte
}

func New(profileUCase ProfileUsecase, secret []byte) *HandlerLogin {
	return &HandlerLogin{
		profileUCase: profileUCase,
		JWTSecret:    secret,
	}
}

func (h *HandlerLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("handle /auth/login")

	defer func() {
		if r := recover(); r != nil {
			log.Error("panic recovered", slog.String("recover", fmt.Sprintf("%v", r)))
			resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, "Некорректный формат запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req.Login = strings.TrimSpace(req.Login)
	req.Password = strings.TrimSpace(req.Password)

	username, err := h.validateRequest(req.Login, req.Password)
	if err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	loginReq := profile.LoginRequest{
		Username: username,
		Password: req.Password,
	}

	userID, err := h.profileUCase.Login(r.Context(), loginReq)
	if err != nil {
		log.Warn("login failed", slog.String("username", username))
		resp.SendErrorResponse(w, "Неверный логин или пароль", http.StatusBadRequest)
		return
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    "access",
	})

	accessTokenString, err := accessToken.SignedString(h.JWTSecret)
	if err != nil {
		log.Error("failed to sign access token")
		resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
		return
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":    "refresh",
	})

	refreshTokenString, err := refreshToken.SignedString(h.JWTSecret)
	if err != nil {
		log.Error("failed to sign refresh token")
		resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
		return
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessTokenString,
		MaxAge:   15 * 60,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(w, accessCookie)

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshTokenString,
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(w, refreshCookie)

	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "успешно",
			Body:    struct{}{},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response")
		resp.SendErrorResponse(w, "Не удалось сформировать ответ", http.StatusInternalServerError)
		return
	}
}

func (h *HandlerLogin) validateRequest(login, password string) (string, error) {
	if login == "" || password == "" {
		return "", fmt.Errorf("Логин и пароль обязательны")
	}

	username := login
	if strings.Contains(login, "@") {
		parts := strings.Split(login, "@")
		if len(parts) > 0 && parts[0] != "" {
			username = strings.TrimSpace(parts[0])
		} else {
			return "", fmt.Errorf("Некорректный логин или email")
		}
	}

	if len(username) < 3 || len(username) > 50 {
		return "", fmt.Errorf("Логин должен быть от 3 до 50 символов")
	}

	for _, char := range username {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return "", fmt.Errorf("Логин может содержать только буквы, цифры и символ подчеркивания")
		}
	}

	if valid.HasDangerousCharacters(username) {
		return "", fmt.Errorf("Логин содержит недопустимые символы")
	}

	if len(password) < 6 {
		return "", fmt.Errorf("Пароль должен быть не короче 6 символов")
	}

	if valid.HasDangerousCharacters(password) {
		return "", fmt.Errorf("Пароль содержит недопустимые символы")
	}

	return username, nil
}
