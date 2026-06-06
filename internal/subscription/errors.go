package subscription

import (
	"errors"

	"GithubReleaseNotificationAPI/internal/subscription/internal/domain"
)

var (
	ErrInvalidEmailFormat        = domain.ErrInvalidEmailFormat
	ErrInvalidRepoFormat         = errors.New("invalid repository format, has to be owner/repo")
	ErrTokenNotFound             = errors.New("token not found")
	ErrRepoNotFound              = errors.New("repository not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription with such email and repo pair already exists")
	ErrTooMuchRequests           = errors.New("request limit hit")
	ErrGitHubUnauthorized        = errors.New("github API token is invalid or expired")
)
