package handler

import (
	"GithubReleaseNotificationAPI/internal/http/models"
	"GithubReleaseNotificationAPI/internal/http/util"
	"GithubReleaseNotificationAPI/internal/service"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	subscriptionService service.SubscriptionService
}

func New(subscriptionService service.SubscriptionService) *Handler {
	return &Handler{
		subscriptionService: subscriptionService,
	}
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	req, err := decodeSubscriptionRequest(r)
	if err != nil {
		util.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Email == "" || req.Repo == "" {
		util.WriteErrorResponse(w, http.StatusBadRequest, "email/repo is empty")
		return
	}

	if err := h.subscriptionService.Subscribe(r.Context(), req.Email, req.Repo); err != nil {
		handleError(w, err)
		return
	}

	util.WriteJSONResponse(w, http.StatusOK, map[string]string{
		"message": "Subscription successful. Confirmation email sent",
	})
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if token == "" {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	err := h.subscriptionService.Confirm(r.Context(), token)
	if err != nil {
		handleError(w, err)
		return
	}

	util.WriteJSONResponse(w, http.StatusOK, map[string]string{
		"message": "Subscription confirmed successfully"})
}

func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if token == "" || len(token) < 8 {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	err := h.subscriptionService.Unsubscribe(r.Context(), token)
	if err != nil {
		handleError(w, err)
		return
	}
	util.WriteJSONResponse(w, http.StatusOK, map[string]string{
		"message": "Unsubscribed successfully",
	})
}

func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	if email == "" {
		util.WriteErrorResponse(w, http.StatusBadRequest, "empty email")
		return
	}

	subscriptions, err := h.subscriptionService.ListByEmail(r.Context(), email)
	if err != nil {
		handleError(w, err)
		return
	}

	util.WriteJSONResponse(w, http.StatusOK, models.ConvertToResponseModel(subscriptions))
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
