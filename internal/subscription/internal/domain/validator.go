package domain

import (
	"errors"
	"net/mail"
	"strings"
)

var (
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrInvalidRepoFormat  = errors.New("invalid repository format, has to be owner/repo")
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
