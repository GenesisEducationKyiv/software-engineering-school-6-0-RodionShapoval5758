package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"GithubReleaseNotificationAPI/internal/domain"
	"GithubReleaseNotificationAPI/internal/http/models"
	"GithubReleaseNotificationAPI/internal/http/respond"

	"github.com/go-chi/chi/v5"
)

type subscriptionService interface {
	Subscribe(ctx context.Context, email string, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	ListByEmail(ctx context.Context, email string) ([]domain.SubscriptionDetails, error)
}

type Handler struct {
	subscriptionService subscriptionService
}

func New(subscriptionService subscriptionService) *Handler {
	return &Handler{
		subscriptionService: subscriptionService,
	}
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

	if err := h.subscriptionService.Subscribe(r.Context(), req.Email, req.Repo); err != nil {
		handleError(w, err)

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

	if err := h.subscriptionService.Confirm(r.Context(), token); err != nil {
		handleError(w, err)

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

	if err := h.subscriptionService.Unsubscribe(r.Context(), token); err != nil {
		handleError(w, err)

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

	subscriptions, err := h.subscriptionService.ListByEmail(r.Context(), email)
	if err != nil {
		handleError(w, err)

		return
	}

	respond.JSON(w, http.StatusOK, models.ConvertToResponseModel(subscriptions))
}

func (h *Handler) ValidateAPIKey(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func decodeSubscriptionRequest(r *http.Request) (models.SubscriptionRequest, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var req models.SubscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return models.SubscriptionRequest{}, err
		}

		return req, nil
	}

	if err := r.ParseForm(); err != nil {
		return models.SubscriptionRequest{}, err
	}

	return models.SubscriptionRequest{
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
