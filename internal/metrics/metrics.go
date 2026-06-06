package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	DBPoolTotalConns    prometheus.Gauge
	DBPoolAcquiredConns prometheus.Gauge
	DBPoolIdleConns     prometheus.Gauge
	DBPoolMaxConns      prometheus.Gauge

	DBPoolEmptyAcquireTotal     prometheus.Counter
	DBPoolEmptyAcquireWaitTotal prometheus.Counter
	DBPoolCanceledAcquireTotal  prometheus.Counter

	WatcherScansTotal   *prometheus.CounterVec
	WatcherScanDuration prometheus.Histogram

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

		DBPoolTotalConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_total_connections",
			Help: "Total number of connections currently in the pool (acquired + idle + constructing).",
		}),
		DBPoolAcquiredConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_acquired_connections",
			Help: "Number of connections currently held by goroutines.",
		}),
		DBPoolIdleConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Number of idle connections in the pool.",
		}),
		DBPoolMaxConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_max_connections",
			Help: "Maximum number of connections allowed in the pool.",
		}),

		DBPoolEmptyAcquireTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_pool_empty_acquire_total",
			Help: "Cumulative number of acquires that waited because the pool was empty.",
		}),
		DBPoolEmptyAcquireWaitTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_pool_empty_acquire_wait_seconds_total",
			Help: "Cumulative time in seconds spent waiting for a connection when the pool was empty.",
		}),
		DBPoolCanceledAcquireTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_pool_canceled_acquire_total",
			Help: "Cumulative number of acquires canceled due to context timeout or cancellation.",
		}),

		WatcherScansTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "watcher_scans_total",
				Help: "Total number of watcher scan cycles, partitioned by result.",
			},
			[]string{"result"},
		),

		WatcherScanDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "watcher_scan_duration_seconds",
			Help:    "Duration of a single watcher scan cycle in seconds.",
			Buckets: []float64{1, 5, 10, 30, 60},
		}),
	}

	reg.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.DBPoolTotalConns,
		m.DBPoolAcquiredConns,
		m.DBPoolIdleConns,
		m.DBPoolMaxConns,
		m.DBPoolEmptyAcquireTotal,
		m.DBPoolEmptyAcquireWaitTotal,
		m.DBPoolCanceledAcquireTotal,
		m.WatcherScansTotal,
		m.WatcherScanDuration,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return m
}

func (m *Metrics) CollectDBStats(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var prevEmptyAcquire, prevCanceled int64
	var prevEmptyWait time.Duration

	for {
		select {
		case <-ticker.C:
			stat := pool.Stat()

			m.DBPoolTotalConns.Set(float64(stat.TotalConns()))
			m.DBPoolAcquiredConns.Set(float64(stat.AcquiredConns()))
			m.DBPoolIdleConns.Set(float64(stat.IdleConns()))
			m.DBPoolMaxConns.Set(float64(stat.MaxConns()))

			emptyAcquire := stat.EmptyAcquireCount()
			m.DBPoolEmptyAcquireTotal.Add(float64(emptyAcquire - prevEmptyAcquire))
			prevEmptyAcquire = emptyAcquire

			emptyWait := stat.EmptyAcquireWaitTime()
			m.DBPoolEmptyAcquireWaitTotal.Add((emptyWait - prevEmptyWait).Seconds())
			prevEmptyWait = emptyWait

			canceled := stat.CanceledAcquireCount()
			m.DBPoolCanceledAcquireTotal.Add(float64(canceled - prevCanceled))
			prevCanceled = canceled

		case <-ctx.Done():
			return
		}
	}
}
func (m *Metrics) ObserveScanDuration(seconds float64) {
	m.WatcherScanDuration.Observe(seconds)
}

func (m *Metrics) IncScanResult(result string) {
	m.WatcherScansTotal.WithLabelValues(result).Inc()
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}
