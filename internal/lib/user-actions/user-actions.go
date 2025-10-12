package user_actions

import (
	md "2025_2_a4code/mocks/mock-data"
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

func getUserLogin(r *http.Request, SECRET []byte) (string, error) {
	claims, err := CheckSession(r, SECRET)
	if err != nil {
		return "", err
	}
	login, ok := claims["login"].(string)
	if !ok {
		return "", fmt.Errorf("логин не найден в токене")
	}

	return login, nil
}

func GetCurrentUserData(r *http.Request, SECRET []byte, users md.MockDataSignup) (map[string]string, error) {
	login, err := getUserLogin(r, SECRET)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user["login"] == login {
			return user, nil
		}
	}

	return nil, fmt.Errorf("пользователь не найден")
}
