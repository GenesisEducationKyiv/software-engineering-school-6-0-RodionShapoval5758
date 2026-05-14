package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(bytes)

	return token, nil
}
