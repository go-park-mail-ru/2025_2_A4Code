package rand

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRandID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
