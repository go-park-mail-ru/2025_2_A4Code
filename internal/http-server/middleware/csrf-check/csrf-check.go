package csrf_check

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
)

func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func New() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(csrfCookieName)
			clientCookieToken := ""
			if err == nil {
				clientCookieToken = cookie.Value
			}

			clientHeaderToken := r.Header.Get(csrfHeaderName)

			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				if clientCookieToken == "" {
					newToken, err := generateToken()
					if err != nil {
						slog.Error(err.Error())
					}
					http.SetCookie(w, &http.Cookie{
						Name:     csrfCookieName,
						Value:    newToken,
						Path:     "/",
						Secure:   true,
						HttpOnly: true,
						SameSite: http.SameSiteNoneMode,
					})
					clientCookieToken = newToken
				}

				next.ServeHTTP(w, r)
				return
			}

			if clientCookieToken == "" || clientHeaderToken == "" || !strings.EqualFold(clientCookieToken, clientHeaderToken) {
				resp.SendErrorResponse(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
