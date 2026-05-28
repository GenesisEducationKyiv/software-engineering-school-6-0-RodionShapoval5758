package router

import (
	"net/http"

	"GithubReleaseNotificationAPI/internal/http/handler"
	"GithubReleaseNotificationAPI/internal/http/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func New(handler *handler.Handler, apiKey string) http.Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)

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
