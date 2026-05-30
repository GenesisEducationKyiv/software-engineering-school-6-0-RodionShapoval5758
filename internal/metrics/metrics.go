package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	reg *prometheus.Registry
}


func New(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		reg: reg,

		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests, partitioned by method, path, and status code.",
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request latency in seconds, partitioned by method and path.",

				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}

	reg.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}
