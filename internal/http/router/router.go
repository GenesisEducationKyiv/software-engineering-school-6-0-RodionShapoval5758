package router

import (
	"net/http"

	"GithubReleaseNotificationAPI/internal/http/handler"
	"GithubReleaseNotificationAPI/internal/http/middleware"
	"GithubReleaseNotificationAPI/internal/metrics"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func New(handler *handler.Handler, apiKey string, m *metrics.Metrics) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.SkipRoutes(middleware.Logger, "/metrics", "/health"))
	router.Use(middleware.SkipRoutes(middleware.MetricsMiddleware(m), "/metrics"))
	router.Use(chimiddleware.Recoverer)

	router.Handle("/metrics", m.Handler())
	router.Get("/health", handler.Health)

	if apiKey != "" {
		router.Route("/api", func(r chi.Router) {
			r.Use(middleware.AuthAPIKEY(apiKey))
			r.Post("/subscribe", handler.Subscribe)
			r.Get("/subscriptions", handler.ListSubscriptions)
			r.Get("/validate", handler.ValidateAPIKey)
		})
	} else {
		router.Get("/api/subscriptions", handler.ListSubscriptions)
		router.Post("/api/subscribe", handler.Subscribe)
	}

	router.Get("/api/unsubscribe/{token}", handler.Unsubscribe)
	router.Get("/api/confirm/{token}", handler.Confirm)

	return router
}
