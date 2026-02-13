package middleware

import "github.com/prometheus/client_golang/prometheus"

// Metrics defines behavioral methods for observing sentinel request handling.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordRequest(statusCode, errorType, environmentID, region string, duration float64)
	IncActiveRequests(environmentID, region string)
	DecActiveRequests(environmentID, region string)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordRequest(string, string, string, string, float64) {}
func (NoopMetrics) IncActiveRequests(string, string)                      {}
func (NoopMetrics) DecActiveRequests(string, string)                      {}

type prometheusMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	activeRequests  *prometheus.GaugeVec
}

func (m *prometheusMetrics) RecordRequest(statusCode, errorType, environmentID, region string, duration float64) {
	m.requestsTotal.WithLabelValues(statusCode, errorType, environmentID, region).Inc()
	m.requestDuration.WithLabelValues(statusCode, errorType, environmentID, region).Observe(duration)
}

func (m *prometheusMetrics) IncActiveRequests(environmentID, region string) {
	m.activeRequests.WithLabelValues(environmentID, region).Inc()
}

func (m *prometheusMetrics) DecActiveRequests(environmentID, region string) {
	m.activeRequests.WithLabelValues(environmentID, region).Dec()
}

// NewPrometheusMetrics creates and registers sentinel middleware metrics on the given registry.
func NewPrometheusMetrics(reg prometheus.Registerer) Metrics {
	m := &prometheusMetrics{
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sentinel_requests_total",
			Help: "Total number of requests processed by sentinel",
		}, []string{"status_code", "error_type", "environment_id", "region"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "sentinel_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"status_code", "error_type", "environment_id", "region"}),
		activeRequests: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sentinel_active_requests",
			Help: "Number of requests currently being processed",
		}, []string{"environment_id", "region"}),
	}

	reg.MustRegister(m.requestsTotal)
	reg.MustRegister(m.requestDuration)
	reg.MustRegister(m.activeRequests)

	return m
}
