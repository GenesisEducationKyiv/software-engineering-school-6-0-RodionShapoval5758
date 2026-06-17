package handler

import (
	"context"
	"net/http"
	"strconv"

	"GithubReleaseNotificationAPI/internal/subscription"
	"GithubReleaseNotificationAPI/internal/transport/http/respond"

	"github.com/go-chi/chi/v5"
)

type confirmedSubsService interface {
	ListConfirmedByRepositoryID(ctx context.Context, repositoryID int64) ([]subscription.Subscription, error)
}

type InternalHandler struct {
	svc   confirmedSubsService
	token string
}

func NewInternal(svc confirmedSubsService, internalToken string) *InternalHandler {
	return &InternalHandler{svc: svc, token: internalToken}
}

type confirmedSubscriberResponse struct {
	Email            string `json:"email"`
	UnsubscribeToken string `json:"unsubscribe_token"`
}

func (h *InternalHandler) ListConfirmedByRepositoryID(w http.ResponseWriter, r *http.Request) {
	if h.token != "" && r.Header.Get("X-Internal-Token") != h.token {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rawID := chi.URLParam(r, "id")

	repoID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || repoID <= 0 {
		respond.Error(w, http.StatusBadRequest, "invalid repository id")
		return
	}

	subs, err := h.svc.ListConfirmedByRepositoryID(r.Context(), repoID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]confirmedSubscriberResponse, len(subs))
	for i, s := range subs {
		resp[i] = confirmedSubscriberResponse{
			Email:            s.Email,
			UnsubscribeToken: s.UnsubscribeToken,
		}
	}

	respond.JSON(w, http.StatusOK, resp)
}
