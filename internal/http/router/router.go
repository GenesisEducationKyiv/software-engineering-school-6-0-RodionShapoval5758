package router

import (
	"encoding/json"
	"net/http"

	"GithubReleaseNotificationAPI/internal/http/middleware"
	"GithubReleaseNotificationAPI/internal/metrics"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type subscriptionHandler interface {
	Subscribe(http.ResponseWriter, *http.Request)
	Confirm(http.ResponseWriter, *http.Request)
	Unsubscribe(http.ResponseWriter, *http.Request)
	ListSubscriptions(http.ResponseWriter, *http.Request)
	ValidateAPIKey(http.ResponseWriter, *http.Request)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func New(handler subscriptionHandler, apiKey string, m *metrics.Metrics) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.SkipRoutes(middleware.Logger, "/metrics", "/health"))
	r.Use(middleware.SkipRoutes(middleware.MetricsMiddleware(m), "/metrics"))
	r.Use(chimiddleware.Recoverer)

	r.Handle("/metrics", m.Handler())
	r.Get("/health", health)

	if apiKey != "" {
		r.Route("/api", func(r chi.Router) {
			r.Use(middleware.AuthAPIKEY(apiKey))
			r.Post("/subscribe", handler.Subscribe)
			r.Get("/subscriptions", handler.ListSubscriptions)
			r.Get("/validate", handler.ValidateAPIKey)
		})
	} else {
		r.Get("/api/subscriptions", handler.ListSubscriptions)
		r.Post("/api/subscribe", handler.Subscribe)
	}

	r.Get("/api/unsubscribe/{token}", handler.Unsubscribe)
	r.Get("/api/confirm/{token}", handler.Confirm)

	return r
}
