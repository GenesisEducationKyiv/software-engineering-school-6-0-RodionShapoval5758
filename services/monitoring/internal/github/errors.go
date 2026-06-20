package github

import (
	"errors"
	"fmt"

	"GithubReleaseNotificationAPI/services/monitoring/internal/db"
)

var (
	ErrNotFound           = fmt.Errorf("repository not found: %w", db.ErrNotFound)
	ErrRateLimited        = errors.New("rate limited")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrUnexpectedResponse = errors.New("unexpected github API response")
)
