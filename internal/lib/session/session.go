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
	ErrorWrongTokenType  = errors.New("wrong token type")
)

func CheckSessionWithToken(r *http.Request, SECRET []byte, cookieName, expectedType string) (jwt.MapClaims, error) {
	const op = "session.CheckSessionWithToken"

	cookie, err := r.Cookie(cookieName)
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

	// Проверяем тип токена, если указан
	if expectedType != "" {
		if tokenType, ok := claims["type"].(string); !ok || tokenType != expectedType {
			return jwt.MapClaims{}, ErrorWrongTokenType
		}
	}

	return claims, nil
}

// Т.к. во всех handlers кроме refresh используем эту функцию, то название упрощено
func CheckSession(r *http.Request, SECRET []byte) (jwt.MapClaims, error) {
	return CheckSessionWithToken(r, SECRET, "access_token", "access")
}

func CheckSessionWithRefreshToken(r *http.Request, SECRET []byte) (jwt.MapClaims, error) {
	return CheckSessionWithToken(r, SECRET, "refresh_token", "refresh")
}

func GetProfileIDFromToken(r *http.Request, SECRET []byte, cookieName, expectedType string) (int64, error) {
	claims, err := CheckSessionWithToken(r, SECRET, cookieName, expectedType)
	if err != nil {
		return -1, err
	}

	id, ok := claims["user_id"].(float64)
	if !ok {
		return -1, ErrorIdNotFound
	}

	return int64(id), nil
}

// Т.к. во всех handlers кроме refresh используем эту функцию, то название упрощено
func GetProfileID(r *http.Request, SECRET []byte) (int64, error) {
	return GetProfileIDFromToken(r, SECRET, "access_token", "access")
}

func GetProfileIDFromRefresh(r *http.Request, SECRET []byte) (int64, error) {
	return GetProfileIDFromToken(r, SECRET, "refresh_token", "refresh")
}
