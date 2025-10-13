package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func CheckSession(r *http.Request, SECRET []byte) (jwt.MapClaims, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return jwt.MapClaims{}, fmt.Errorf("сессия не найдена")
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return SECRET, nil
	})

	if err != nil || !token.Valid {
		return jwt.MapClaims{}, fmt.Errorf("неверный токен")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return jwt.MapClaims{}, fmt.Errorf("ошибка при чтении данных токена")
	}

	// Проверяем expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return jwt.MapClaims{}, fmt.Errorf("токен просрочен")
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
		return -1, fmt.Errorf("id не найден в токене")
	}

	return id, nil
}
