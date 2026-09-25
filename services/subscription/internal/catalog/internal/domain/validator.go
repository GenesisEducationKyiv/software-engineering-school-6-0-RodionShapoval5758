package domain

import (
	"errors"
	"strings"
)

var ErrInvalidRepoFormat = errors.New("invalid repository format, has to be owner/repo")

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
