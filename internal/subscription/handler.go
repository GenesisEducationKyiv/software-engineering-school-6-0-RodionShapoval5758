package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"GithubReleaseNotificationAPI/internal/http/middleware"
	"GithubReleaseNotificationAPI/internal/http/respond"

	"github.com/go-chi/chi/v5"
)

type subscriptionService interface {
	Subscribe(ctx context.Context, email string, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	ListByEmail(ctx context.Context, email string) ([]SubscriptionDetails, error)
}

type Handler struct {
	svc subscriptionService
}

func New(svc subscriptionService) *Handler {
	return &Handler{svc: svc}
}

type subscriptionRequest struct {
	Email string `json:"email"`
	Repo  string `json:"repo"`
}

type subscriptionResponse struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag"`
}

func toResponseSlice(details []SubscriptionDetails) []subscriptionResponse {
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
	req, err := decodeSubscriptionRequest(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	if err := requireNonEmptySubscriptionFields(req.Email, req.Repo); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	if err := h.svc.Subscribe(r.Context(), req.Email, req.Repo); err != nil {
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

	if err := h.svc.Confirm(r.Context(), token); err != nil {
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

	if err := h.svc.Unsubscribe(r.Context(), token); err != nil {
		handleError(w, r, err)

		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{
		"message": "Unsubscribed successfully",
	})
}

func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	if err := requireNonEmptyEmail(email); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	details, err := h.svc.ListByEmail(r.Context(), email)
	if err != nil {
		handleError(w, r, err)

		return
	}

	respond.JSON(w, http.StatusOK, toResponseSlice(details))
}

func (h *Handler) ValidateAPIKey(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
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
		Email: r.Form.Get("email"),
		Repo:  r.Form.Get("repo"),
	}, nil
}

func requireNonEmptySubscriptionFields(email, repo string) error {
	if email == "" {
		return errors.New("email is empty")
	}

	if repo == "" {
		return errors.New("repo is empty")
	}

	return nil
}

func requireToken(token string, minLen int) error {
	if len(token) < minLen {
		return errors.New("invalid token")
	}

	return nil
}

func requireNonEmptyEmail(email string) error {
	if email == "" {
		return errors.New("empty email")
	}

	return nil
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmailFormat):
		respond.Error(w, http.StatusBadRequest, "Invalid email format")
	case errors.Is(err, ErrInvalidRepoFormat):
		respond.Error(w, http.StatusBadRequest, "Invalid repo format")
	case errors.Is(err, ErrTokenNotFound):
		respond.Error(w, http.StatusNotFound, "Token not found")
	case errors.Is(err, ErrRepoNotFound):
		respond.Error(w, http.StatusNotFound, "Repository not found on GitHub")
	case errors.Is(err, ErrSubscriptionAlreadyExists):
		respond.Error(w, http.StatusConflict, "Email already subscribed to this repository")
	case errors.Is(err, ErrTooMuchRequests):
		respond.Error(w, http.StatusTooManyRequests, "Github API request limit is hit")
	case errors.Is(err, ErrGitHubUnauthorized):
		respond.Error(w, http.StatusBadGateway, "GitHub API token is invalid or expired")
	default:
		middleware.LoggerFromContext(r.Context()).Error("internal server error", "error", err.Error())
		respond.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
