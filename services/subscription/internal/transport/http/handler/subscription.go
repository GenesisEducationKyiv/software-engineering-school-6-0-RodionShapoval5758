package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/middleware"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/respond"

	"github.com/go-chi/chi/v5"
)

type subscriber interface {
	Execute(ctx context.Context, email, repo string) error
}

type confirmer interface {
	Execute(ctx context.Context, token string) error
}

type unsubscriber interface {
	Execute(ctx context.Context, token string) error
}

type lister interface {
	ByEmail(ctx context.Context, email string) ([]subscription.SubscriptionDetails, error)
}

type Handler struct {
	subscriber   subscriber
	confirmer    confirmer
	unsubscriber unsubscriber
	lister       lister
}

func New(sub subscriber, conf confirmer, unsub unsubscriber, list lister) *Handler {
	return &Handler{
		subscriber:   sub,
		confirmer:    conf,
		unsubscriber: unsub,
		lister:       list,
	}
}

type subscriptionRequest struct {
	Repo string `json:"repo"`
}

type subscriptionResponse struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag"`
}

func toResponseSlice(details []subscription.SubscriptionDetails) []subscriptionResponse {
	responses := make([]subscriptionResponse, 0, len(details))
	for _, d := range details {
		tag := d.LastSeenTag
		if tag == "" {
			tag = "not available yet"
		}

		responses = append(responses, subscriptionResponse{
			Email:       d.Email,
			Repo:        d.Repo,
			Confirmed:   d.Confirmed,
			LastSeenTag: tag,
		})
	}

	return responses
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	email := middleware.EmailFromContext(r.Context())
	if email == "" {
		respond.Error(w, http.StatusUnauthorized, "Not authorized")

		return
	}

	req, err := decodeSubscriptionRequest(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	if req.Repo == "" {
		respond.Error(w, http.StatusBadRequest, "repo is required")

		return
	}

	if err := h.subscriber.Execute(r.Context(), email, req.Repo); err != nil {
		handleError(w, r, err)

		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{
		"message": "Subscription successful. Confirmation email sent",
	})
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if err := requireToken(token, 1); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	if err := h.confirmer.Execute(r.Context(), token); err != nil {
		handleError(w, r, err)

		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{
		"message": "Subscription confirmed successfully",
	})
}

func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if err := requireToken(token, 8); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	if err := h.unsubscriber.Execute(r.Context(), token); err != nil {
		handleError(w, r, err)

		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{
		"message": "Unsubscribed successfully",
	})
}

func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	email := middleware.EmailFromContext(r.Context())
	if email == "" {
		respond.Error(w, http.StatusUnauthorized, "Not authorized")

		return
	}

	details, err := h.lister.ByEmail(r.Context(), email)
	if err != nil {
		handleError(w, r, err)

		return
	}

	respond.JSON(w, http.StatusOK, toResponseSlice(details))
}

func decodeSubscriptionRequest(r *http.Request) (subscriptionRequest, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var req subscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return subscriptionRequest{}, err
		}

		return req, nil
	}

	if err := r.ParseForm(); err != nil {
		return subscriptionRequest{}, err
	}

	return subscriptionRequest{
		Repo: r.Form.Get("repo"),
	}, nil
}

func requireToken(token string, minLen int) error {
	if len(token) < minLen {
		return errors.New("invalid token")
	}

	return nil
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, subscription.ErrInvalidEmailFormat):
		respond.Error(w, http.StatusBadRequest, "Invalid email format")
	case errors.Is(err, subscription.ErrInvalidRepoFormat):
		respond.Error(w, http.StatusBadRequest, "Invalid repo format")
	case errors.Is(err, subscription.ErrTokenNotFound):
		respond.Error(w, http.StatusNotFound, "Token not found")
	case errors.Is(err, subscription.ErrRepoNotFound):
		respond.Error(w, http.StatusNotFound, "Repository not found on GitHub")
	case errors.Is(err, subscription.ErrSubscriptionAlreadyExists):
		respond.Error(w, http.StatusConflict, "Email already subscribed to this repository")
	case errors.Is(err, subscription.ErrTooMuchRequests):
		respond.Error(w, http.StatusTooManyRequests, "Github API request limit is hit")
	case errors.Is(err, subscription.ErrGitHubUnauthorized):
		respond.Error(w, http.StatusBadGateway, "GitHub API token is invalid or expired")
	default:
		middleware.LoggerFromContext(r.Context()).Error("internal server error", "error", err.Error())
		respond.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
