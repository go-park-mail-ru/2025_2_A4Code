package signup

import (
	resp "2025_2_a4code/internal/lib/api/response"

	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Request struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Birthday string `json:"birthday"`
	Gender   string `json:"gender"`
	Password string `json:"password"`
}

type Response struct {
	resp.Response
}

type HandlerSignup struct {
	profileUCase *profileUcase.ProfileUcase
	log          *slog.Logger
	JWTSecret    []byte
}

func New(ucP *profileUcase.ProfileUcase, log *slog.Logger, secret []byte) *HandlerSignup {
	return &HandlerSignup{
		profileUCase: ucP,
		log:          log,
		JWTSecret:    secret,
	}
}

func (h *HandlerSignup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle /auth/signup")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, "invalid request format", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	req.Birthday = strings.TrimSpace(req.Birthday)
	req.Gender = strings.TrimSpace(req.Gender)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" || req.Name == "" || req.Gender == "" || req.Birthday == "" {
		resp.SendErrorResponse(w, "all form fields are required", http.StatusBadRequest)
		return
	}

	// Преобразуем в UseCase запрос
	SignupReq := profileUcase.SignupRequest{
		Name:     req.Name,
		Username: req.Username,
		Birthday: req.Birthday,
		Gender:   req.Gender,
		Password: req.Password,
	}

	userID, err := h.profileUCase.Signup(r.Context(), SignupReq)
	if err != nil {
		log.Warn("signup failed", slog.String("username", req.Username))
		resp.SendErrorResponse(w, "signup failed", http.StatusBadRequest)
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
			Message: "signup successful",
			Body:    struct{}{},
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
