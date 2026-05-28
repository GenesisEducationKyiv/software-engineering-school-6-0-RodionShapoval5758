package domain

import (
	"net/mail"
	"strings"
)

func ValidateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmailFormat
	}

	return nil
}

func ValidateRepo(repo string) error {
	parts := strings.Split(repo, "/")

	if len(parts) != 2 {
		return ErrInvalidRepoFormat
	}

	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return ErrInvalidRepoFormat
	}

	return nil
}
