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
		resp.SendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, "invalid request format", http.StatusBadRequest)
		return
	}

	req.Login = strings.TrimSpace(req.Login)
	req.Password = strings.TrimSpace(req.Password)

	if req.Login == "" || req.Password == "" {
		resp.SendErrorResponse(w, "all form fields are required", http.StatusBadRequest)
		return
	}

	username := req.Login
	if strings.Contains(req.Login, "@") {
		parts := strings.Split(req.Login, "@")
		if len(parts) > 0 && parts[0] != "" {
			username = strings.TrimSpace(parts[0])
		} else {
			resp.SendErrorResponse(w, "invalid login or email format", http.StatusBadRequest)
			return
		}
	}

	// Преобразуем в UseCase запрос
	LoginReq := profileUcase.LoginRequest{
		Username: username,
		Password: req.Password,
	}

	userID, err := h.profileUCase.Login(r.Context(), LoginReq)
	if err != nil {
		log.Warn("login failed", slog.String("username", username))
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
		log.Error("failed to sign resfresh token")
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
			Status:  http.StatusText(http.StatusOK),
			Message: "login successful",
			Body:    struct{}{},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
