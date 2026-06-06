package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"GithubReleaseNotificationAPI/internal/idgen"

	"github.com/go-chi/chi/v5"
)

type contextKey struct{}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return l
	}

	slog.Default().Warn("LoggerFromContext: no logger in context, request_id will be missing")

	return slog.Default()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := idgen.New()
		logger := slog.Default().With("request_id", requestID)
		ctx := context.WithValue(r.Context(), contextKey{}, logger)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(rec, r.WithContext(ctx))

		path := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil {
			if p := rc.RoutePattern(); p != "" {
				path = p
			}
		}

		logger.Info("http request",
			"method", r.Method,
			"path", path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
