package middleware

import (
	"net/http"
	"strconv"
	"time"

	"GithubReleaseNotificationAPI/internal/metrics"

	"github.com/go-chi/chi/v5"
)

const unmatchedRoute = "unmatched"

func MetricsMiddleware(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(rec, r)

			pattern := unmatchedRoute
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if p := rc.RoutePattern(); p != "" {
					pattern = p
				}
			}

			status := strconv.Itoa(rec.status)
			elapsed := time.Since(start).Seconds()

			m.HTTPRequestsTotal.WithLabelValues(r.Method, pattern, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(r.Method, pattern).Observe(elapsed)
		})
	}
}
