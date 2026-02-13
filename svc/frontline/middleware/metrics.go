package middleware

import "github.com/prometheus/client_golang/prometheus"

// Metrics defines behavioral methods for observing frontline request handling.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordRequest(statusCode, errorType, region string, duration float64)
	IncActiveRequests(region string)
	DecActiveRequests(region string)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordRequest(string, string, string, float64) {}
func (NoopMetrics) IncActiveRequests(string)                      {}
func (NoopMetrics) DecActiveRequests(string)                      {}

type prometheusMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	activeRequests  *prometheus.GaugeVec
}

func (m *prometheusMetrics) RecordRequest(statusCode, errorType, region string, duration float64) {
	m.requestsTotal.WithLabelValues(statusCode, errorType, region).Inc()
	m.requestDuration.WithLabelValues(statusCode, errorType, region).Observe(duration)
}

func (m *prometheusMetrics) IncActiveRequests(region string) {
	m.activeRequests.WithLabelValues(region).Inc()
}

func (m *prometheusMetrics) DecActiveRequests(region string) {
	m.activeRequests.WithLabelValues(region).Dec()
}

// NewPrometheusMetrics creates and registers frontline middleware metrics on the given registry.
func NewPrometheusMetrics(reg prometheus.Registerer) Metrics {
	m := &prometheusMetrics{
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "frontline_requests_total",
			Help: "Total number of requests processed by frontline",
		}, []string{"status_code", "error_type", "region"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "frontline_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"status_code", "error_type", "region"}),
		activeRequests: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "frontline_active_requests",
			Help: "Number of requests currently being processed",
		}, []string{"region"}),
	}

	reg.MustRegister(m.requestsTotal)
	reg.MustRegister(m.requestDuration)
	reg.MustRegister(m.activeRequests)

	return m
}
