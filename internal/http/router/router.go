package router

import (
	"GithubReleaseNotificationAPI/internal/http/handler"
	"GithubReleaseNotificationAPI/internal/http/middlewaref"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(handler *handler.Handler, apiKey string) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	if apiKey != "" {
		router.Route("/api", func(r chi.Router) {
			r.Use(middlewaref.AuthAPIKEY(apiKey))
			r.Post("/subscribe", handler.Subscribe)
			r.Get("/subscriptions", handler.ListSubscriptions)
		})
	} else  {
		router.Get("/api/subscriptions", handler.ListSubscriptions)
		router.Post("/api/subscribe", handler.Subscribe)
	}

	router.Get("/api/unsubscribe/{token}", handler.Unsubscribe)
	router.Get("/api/confirm/{token}", handler.Confirm)

	return router
}
