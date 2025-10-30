package session

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrorInvalidToken    = errors.New("invalid token")
	ErrorTokenExpired    = errors.New("token expired")
	ErrorSessionNotFound = errors.New("session not found")
	ErrorIdNotFound      = errors.New("id not found")
)

func CheckSession(r *http.Request, SECRET []byte) (jwt.MapClaims, error) {
	const op = "session.CheckSession"

	cookie, err := r.Cookie("session_id")
	if err != nil {
		return jwt.MapClaims{}, ErrorSessionNotFound
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected sign method: %v", token.Header["alg"])
		}
		return SECRET, nil
	})

	if err != nil || !token.Valid {
		return jwt.MapClaims{}, ErrorInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return jwt.MapClaims{}, ErrorSessionNotFound
	}

	// Проверяем expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return jwt.MapClaims{}, ErrorTokenExpired
		}
	}

	return claims, nil
}

func GetProfileID(r *http.Request, SECRET []byte) (int64, error) {
	claims, err := CheckSession(r, SECRET)
	if err != nil {
		return -1, err
	}
	id, ok := claims["ID"].(int64)
	if !ok {
		return -1, ErrorIdNotFound
	}

	return id, nil
}
