package handler

import (
	"GithubReleaseNotificationAPI/internal/http/util"
	"GithubReleaseNotificationAPI/internal/service"
	"errors"
	"log/slog"
	"net/http"
)

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidEmailFormat):
		util.WriteErrorResponse(w, http.StatusBadRequest, "Invalid email format")
	case errors.Is(err, service.ErrInvalidRepoFormat):
		util.WriteErrorResponse(w, http.StatusBadRequest, "Invalid repo format")
	case errors.Is(err, service.ErrTokenNotFound):
		util.WriteErrorResponse(w, http.StatusNotFound, "Token not found")
	case errors.Is(err, service.ErrRepoNotFound):
		util.WriteErrorResponse(w, http.StatusNotFound, "Repository not found on GitHub")
	case errors.Is(err, service.ErrSubscriptionAlreadyExists):
		util.WriteErrorResponse(w, http.StatusConflict, "Email already subscribed to this repository")
	case errors.Is(err, service.ErrTooMuchRequests):
		util.WriteErrorResponse(w, http.StatusTooManyRequests, "Github API request limit is hit")
	default:
		slog.Error("internal server error", "error", err.Error())
		util.WriteErrorResponse(w, http.StatusInternalServerError, "internal server error")
	}
}
