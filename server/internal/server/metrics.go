package server

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registerSafe calls prometheus.Register and ignores AlreadyRegisteredError.
// This prevents panics in tests and is resilient to duplicate registrations.
func registerSafe(c prometheus.Collector) {
	if err := prometheus.Register(c); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}
}

// MetricsCollector holds all Prometheus metrics for the application.
type MetricsCollector struct {
	RequestCount    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge
	QueueDepth      prometheus.GaugeFunc
	DLQDepth        prometheus.GaugeFunc
	WorkerCount     prometheus.GaugeFunc
}

// NewMetricsCollector creates and registers all Prometheus metrics.
func NewMetricsCollector() *MetricsCollector {
	m := &MetricsCollector{
		RequestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifier_http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "notifier_http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		RequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "notifier_http_requests_in_flight",
				Help: "Current number of in-flight HTTP requests.",
			},
		),
	}

	registerSafe(m.RequestCount)
	registerSafe(m.RequestDuration)
	registerSafe(m.RequestsInFlight)

	return m
}

// SetQueueDepth registers a gauge function that reports the email:jobs stream length.
func (m *MetricsCollector) SetQueueDepth(fn func() float64) {
	m.QueueDepth = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "notifier_queue_depth",
			Help: "Current number of pending jobs in the email queue.",
		},
		fn,
	)
registerSafe(m.QueueDepth)
}

// SetDLQDepth registers a gauge function that reports the DLQ stream length.
func (m *MetricsCollector) SetDLQDepth(fn func() float64) {
	m.DLQDepth = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "notifier_dlq_depth",
			Help: "Current number of messages in the dead letter queue.",
		},
		fn,
	)
registerSafe(m.DLQDepth)
}

// SetWorkerCount registers a gauge for the active worker count.
func (m *MetricsCollector) SetWorkerCount(fn func() float64) {
	m.WorkerCount = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "notifier_worker_count",
			Help: "Current number of active workers.",
		},
		fn,
	)
registerSafe(m.WorkerCount)
}

// MetricsMiddleware records request count, duration, and in-flight gauge.
func MetricsMiddleware(m *MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			m.RequestsInFlight.Inc()

			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(sw.status)

			// Use the route pattern from chi context for cleaner path grouping
			path := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					path = pattern
				}
			}

			m.RequestCount.WithLabelValues(r.Method, path, status).Inc()
			m.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
			m.RequestsInFlight.Dec()
		})
	}
}

// MetricsHandler returns the /metrics endpoint handler.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// LogMetrics logs key metrics periodically for debugging.
func LogMetrics() {
	slog.Debug("metrics endpoint available at /metrics")
}
