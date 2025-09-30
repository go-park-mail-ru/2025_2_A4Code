package handlers

import (
	md "2025_2_a4code/handlers/mock-data"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var SECRET = []byte("secret")
var users md.MockDataSignup

type Handlers struct{}

func New() *Handlers {
	return &Handlers{}
}

func (h *Handlers) Reset() {
	users = nil
}

func (h *Handlers) GetUsers() md.MockDataSignup {
	return users
}

func (h *Handlers) SetUsers(newUsers md.MockDataSignup) {
	users = newUsers
}

func (h *Handlers) CreateToken(login string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": login,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})
	return token.SignedString(SECRET)
}
