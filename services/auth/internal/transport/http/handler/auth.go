package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"GithubReleaseNotificationAPI/services/auth/internal/transport/http/respond"
	"GithubReleaseNotificationAPI/services/auth/internal/user"
	"GithubReleaseNotificationAPI/services/auth/internal/user/usecase"

	"github.com/go-chi/chi/v5"
)

type registerer interface {
	Execute(ctx context.Context, email, password string) error
}

type verifier interface {
	Execute(ctx context.Context, token string) error
}

type loginer interface {
	Execute(ctx context.Context, email, password string) (usecase.TokenPair, error)
}

type refresher interface {
	Execute(ctx context.Context, rawToken string) (usecase.TokenPair, error)
}

type logouter interface {
	Execute(ctx context.Context, rawToken string) error
}

type Handler struct {
	registerer registerer
	verifier   verifier
	loginer    loginer
	refresher  refresher
	logouter   logouter
}

func New(reg registerer, ver verifier, log loginer, ref refresher, out logouter) *Handler {
	return &Handler{
		registerer: reg,
		verifier:   ver,
		loginer:    log,
		refresher:  ref,
		logouter:   out,
	}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")

		return
	}

	if err := h.registerer.Execute(r.Context(), req.Email, req.Password); err != nil {
		handleError(w, err)

		return
	}

	respond.JSON(w, http.StatusCreated, map[string]string{
		"message": "Registered. Check your email to verify your account",
	})
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if len(token) < 8 {
		respond.Error(w, http.StatusBadRequest, "invalid token")

		return
	}

	if err := h.verifier.Execute(r.Context(), token); err != nil {
		handleError(w, err)

		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{
		"message": "Email verified. You can now log in",
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")

		return
	}

	pair, err := h.loginer.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		handleError(w, err)

		return
	}

	respond.JSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRefreshRequest(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	pair, err := h.refresher.Execute(r.Context(), req.RefreshToken)
	if err != nil {
		handleError(w, err)

		return
	}

	respond.JSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRefreshRequest(r)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())

		return
	}

	if err := h.logouter.Execute(r.Context(), req.RefreshToken); err != nil {
		handleError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeRefreshRequest(r *http.Request) (refreshRequest, error) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return refreshRequest{}, errors.New("invalid request body")
	}

	if req.RefreshToken == "" {
		return refreshRequest{}, errors.New("refresh_token is required")
	}

	return req, nil
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrInvalidEmailFormat):
		respond.Error(w, http.StatusBadRequest, "Invalid email format")
	case errors.Is(err, user.ErrWeakPassword):
		respond.Error(w, http.StatusBadRequest, "Password must be at least 8 characters")
	case errors.Is(err, user.ErrEmailTaken):
		respond.Error(w, http.StatusConflict, "Email is already registered")
	case errors.Is(err, user.ErrInvalidCredentials):
		respond.Error(w, http.StatusUnauthorized, "Invalid email or password")
	case errors.Is(err, user.ErrEmailNotVerified):
		respond.Error(w, http.StatusUnauthorized, "Email is not verified")
	case errors.Is(err, user.ErrRefreshTokenInvalid):
		respond.Error(w, http.StatusUnauthorized, "Invalid refresh token")
	case errors.Is(err, user.ErrTokenNotFound):
		respond.Error(w, http.StatusNotFound, "Token not found")
	case errors.Is(err, user.ErrTokenExpired), errors.Is(err, user.ErrTokenUsed):
		respond.Error(w, http.StatusGone, "Token expired or already used")
	default:
		slog.Error("internal server error", "error", err.Error())
		respond.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
