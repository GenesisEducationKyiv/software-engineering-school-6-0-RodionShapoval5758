package handler

import (
	"errors"
	"net/http"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/http/middleware"
	"GithubReleaseNotificationAPI/internal/http/respond"
	"GithubReleaseNotificationAPI/internal/service"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmailFormat):
		respond.Error(w, http.StatusBadRequest, "Invalid email format")
	case errors.Is(err, domain.ErrInvalidRepoFormat):
		respond.Error(w, http.StatusBadRequest, "Invalid repo format")
	case errors.Is(err, service.ErrTokenNotFound):
		respond.Error(w, http.StatusNotFound, "Token not found")
	case errors.Is(err, service.ErrRepoNotFound):
		respond.Error(w, http.StatusNotFound, "Repository not found on GitHub")
	case errors.Is(err, service.ErrSubscriptionAlreadyExists):
		respond.Error(w, http.StatusConflict, "Email already subscribed to this repository")
	case errors.Is(err, service.ErrTooMuchRequests):
		respond.Error(w, http.StatusTooManyRequests, "Github API request limit is hit")
	case errors.Is(err, service.ErrGitHubUnauthorized):
		respond.Error(w, http.StatusBadGateway, "GitHub API token is invalid or expired")
	default:
		middleware.LoggerFromContext(r.Context()).Error("internal server error", "error", err.Error())
		respond.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
