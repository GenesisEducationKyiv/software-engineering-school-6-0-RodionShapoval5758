package router

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"GithubReleaseNotificationAPI/services/subscription/internal/metrics"
	"GithubReleaseNotificationAPI/services/subscription/internal/transport/http/middleware"

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

type Pinger interface {
	Ping(ctx context.Context) error
}

func healthHandler(db, nats Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		checks := map[string]string{
			"db":   checkDep(ctx, db),
			"nats": checkDep(ctx, nats),
		}

		status := http.StatusOK
		overall := "ok"
		for _, v := range checks {
			if v != "ok" {
				status = http.StatusServiceUnavailable
				overall = "unhealthy"
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": overall,
			"checks": checks,
		})
	}
}

func checkDep(ctx context.Context, p Pinger) string {
	if err := p.Ping(ctx); err != nil {
		return err.Error()
	}
	return "ok"
}

func New(handler subscriptionHandler, apiKey string, m *metrics.Metrics, db, nats Pinger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.SkipRoutes(middleware.Logger, "/metrics", "/health"))
	r.Use(middleware.SkipRoutes(middleware.MetricsMiddleware(m), "/metrics"))
	r.Use(chimiddleware.Recoverer)

	r.Handle("/metrics", m.Handler())
	r.Get("/health", healthHandler(db, nats))

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
