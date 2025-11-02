package refresh

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Response struct {
	resp.Response
}

type HandlerRefresh struct {
	log       *slog.Logger
	JWTSecret []byte
}

func New(log *slog.Logger, secret []byte) *HandlerRefresh {
	return &HandlerRefresh{
		log:       log,
		JWTSecret: secret,
	}
}

func (h *HandlerRefresh) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle /auth/refresh")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := session.GetProfileIDFromRefresh(r, h.JWTSecret)
	if err != nil {
		log.Warn("invalid refresh token", slog.String("error", err.Error()))
		http.SetCookie(w, &http.Cookie{
			Name:   "refresh_token",
			Value:  "",
			MaxAge: -1,
			Path:   "/",
		})
		resp.SendErrorResponse(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    "access",
	})

	newAccessTokenString, err := newAccessToken.SignedString(h.JWTSecret)
	if err != nil {
		log.Error("failed to sign new access token")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    newAccessTokenString,
		MaxAge:   15 * 60,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, accessCookie)

	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(http.StatusOK),
			Message: "token refreshed successfully",
			Body:    struct{}{},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
