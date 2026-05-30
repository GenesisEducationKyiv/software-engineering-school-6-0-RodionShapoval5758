package metrics_test

import (
	"testing"

	"GithubReleaseNotificationAPI/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newTestMetrics(t *testing.T) *metrics.Metrics {
	t.Helper()
	return metrics.New(prometheus.NewRegistry())
}

func TestNew_RegistersWithoutPanic(t *testing.T) {
	// Two separate registries must not panic — the common failure mode with globals.
	newTestMetrics(t)
	newTestMetrics(t)
}

func TestNew_HandlerNotNil(t *testing.T) {
	m := newTestMetrics(t)
	if m.Handler() == nil {
		t.Fatal("expected non-nil metrics handler")
	}
}

func TestHTTPRequestsTotal_Increments(t *testing.T) {
	m := newTestMetrics(t)

	m.HTTPRequestsTotal.WithLabelValues("GET", "/api/subscriptions", "200").Inc()
	m.HTTPRequestsTotal.WithLabelValues("GET", "/api/subscriptions", "200").Inc()
	m.HTTPRequestsTotal.WithLabelValues("POST", "/api/subscribe", "201").Inc()

	got := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/api/subscriptions", "200"))
	if got != 2 {
		t.Errorf("expected 2, got %v", got)
	}

	got = testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("POST", "/api/subscribe", "201"))
	if got != 1 {
		t.Errorf("expected 1, got %v", got)
	}
}

func TestHTTPRequestDuration_ObservesOnce(t *testing.T) {
	m := newTestMetrics(t)

	m.HTTPRequestDuration.WithLabelValues("POST", "/api/subscribe").Observe(0.042)

	// After one Observe() call the vec must contain exactly one label series.
	// testutil.CollectAndCount counts distinct *dto.Metric entries in the collector.
	if n := testutil.CollectAndCount(m.HTTPRequestDuration); n != 1 {
		t.Errorf("expected 1 histogram series, got %d", n)
	}
}
