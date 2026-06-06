package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"GithubReleaseNotificationAPI/internal/http/middleware"
	"GithubReleaseNotificationAPI/internal/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newTestMetrics() *metrics.Metrics {
	return metrics.New(prometheus.NewRegistry())
}

func routeRequest(t *testing.T, m *metrics.Metrics, method, pattern, url string, handler http.HandlerFunc) {
	t.Helper()

	r := chi.NewRouter()
	r.Use(middleware.MetricsMiddleware(m))
	r.Method(method, pattern, handler)

	req := httptest.NewRequest(method, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

func TestMetricsMiddleware_CountsRequest(t *testing.T) {
	m := newTestMetrics()

	routeRequest(t, m, http.MethodGet, "/api/subscriptions", "/api/subscriptions",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
	)

	got := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/api/subscriptions", "200"))
	if got != 1 {
		t.Errorf("expected counter 1, got %v", got)
	}
}

func TestMetricsMiddleware_ErrorStatus(t *testing.T) {
	m := newTestMetrics()

	routeRequest(t, m, http.MethodPost, "/api/subscribe", "/api/subscribe",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	)

	got := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("POST", "/api/subscribe", "500"))
	if got != 1 {
		t.Errorf("expected error counter 1, got %v", got)
	}
}

func TestMetricsMiddleware_UsesRoutePattern_NotRawPath(t *testing.T) {
	m := newTestMetrics()

	for _, token := range []string{"abc123", "xyz999"} {
		routeRequest(t, m, http.MethodGet, "/api/confirm/{token}", "/api/confirm/"+token,
			func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		)
	}

	got := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/api/confirm/{token}", "200"))
	if got != 2 {
		t.Errorf("expected pattern-level counter 2, got %v", got)
	}

	rawABC := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/api/confirm/abc123", "200"))
	if rawABC != 0 {
		t.Errorf("raw path /api/confirm/abc123 must not create a series, got %v", rawABC)
	}
}

func TestMetricsMiddleware_UnmatchedRoute(t *testing.T) {
	m := newTestMetrics()

	r := chi.NewRouter()
	r.Use(middleware.MetricsMiddleware(m))
	r.Get("/api/subscriptions", func(w http.ResponseWriter, _ *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "unmatched", "404"))
	if got != 1 {
		t.Errorf("expected unmatched counter 1, got %v", got)
	}
}

func TestMetricsMiddleware_RecordsDuration(t *testing.T) {
	m := newTestMetrics()

	routeRequest(t, m, http.MethodGet, "/api/subscriptions", "/api/subscriptions",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
	)

	if n := testutil.CollectAndCount(m.HTTPRequestDuration); n != 1 {
		t.Errorf("expected 1 histogram series, got %d", n)
	}
}
