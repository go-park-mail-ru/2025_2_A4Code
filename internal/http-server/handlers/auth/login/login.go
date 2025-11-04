package login

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	valid "2025_2_a4code/internal/lib/validation"
	"log/slog"
	"strings"

	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"unicode"

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
	JWTSecret    []byte
}

func New(ucP *profileUcase.ProfileUcase, secret []byte) *HandlerLogin {
	return &HandlerLogin{
		profileUCase: ucP,
		JWTSecret:    secret,
	}
}

func (h *HandlerLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Info("handle /auth/login")

	defer func() {
		if r := recover(); r != nil {
			log.Error("panic recovered",
				slog.String("recover", fmt.Sprintf("%v", r)))
			resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, "invalid request format", http.StatusBadRequest)
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

	// Преобразуем в UseCase запрос
	LoginReq := profileUcase.LoginRequest{
		Username: username,
		Password: req.Password,
	}

	userID, err := h.profileUCase.Login(r.Context(), LoginReq)
	if err != nil {
		log.Warn("login failed",
			slog.String("username", username))
		resp.SendErrorResponse(w, "invalid login or password", http.StatusBadRequest)
		return
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(), // 15 минут
		"type":    "access",
	})

	accessTokenString, err := accessToken.SignedString(h.JWTSecret)
	if err != nil {
		log.Error("failed to sign access token")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 дней
		"type":    "refresh",
	})

	refreshTokenString, err := refreshToken.SignedString(h.JWTSecret)
	if err != nil {
		log.Error("failed to sign refresh token")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessTokenString,
		MaxAge:   15 * 60, // 15 минут
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, accessCookie)

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshTokenString,
		MaxAge:   7 * 24 * 3600, // 7  дней
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, refreshCookie)

	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    struct{}{},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}

func (h *HandlerLogin) validateRequest(login, password string) (string, error) {
	if login == "" || password == "" {
		return "", fmt.Errorf("all fields are required")
	}

	username := login
	if strings.Contains(login, "@") {
		parts := strings.Split(login, "@")
		if len(parts) > 0 && parts[0] != "" {
			username = strings.TrimSpace(parts[0])
		} else {
			return "", fmt.Errorf("invalid login or email format")
		}
	}

	if len(username) < 3 || len(username) > 50 {
		return "", fmt.Errorf("username must be between 3 and 50 characters")
	}

	for _, char := range username {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return "", fmt.Errorf("username can only contain letters, numbers and underscores")
		}
	}

	if valid.HasDangerousCharacters(username) {
		return "", fmt.Errorf("username contains invalid characters")
	}

	if len(password) < 6 {
		return "", fmt.Errorf("password must be at least 6 characters")
	}

	if valid.HasDangerousCharacters(password) {
		return "", fmt.Errorf("password contains invalid characters")
	}

	return username, nil
}
