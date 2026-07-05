package router

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type authHandler interface {
	Register(http.ResponseWriter, *http.Request)
	VerifyEmail(http.ResponseWriter, *http.Request)
	Login(http.ResponseWriter, *http.Request)
	Refresh(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
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

func New(handler authHandler, db, nats Pinger) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)

	r.Get("/health", healthHandler(db, nats))

	r.Post("/register", handler.Register)
	r.Get("/verify-email/{token}", handler.VerifyEmail)
	r.Post("/login", handler.Login)
	r.Post("/refresh", handler.Refresh)
	r.Post("/logout", handler.Logout)

	return r
}
