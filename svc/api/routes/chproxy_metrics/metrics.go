package chproxyMetrics

import "github.com/prometheus/client_golang/prometheus"

// Metrics defines behavioral methods for observing chproxy operations.
// This interface is shared by all three chproxy handlers (metrics, verifications, ratelimits).
type Metrics interface {
	RecordChproxyRequest(endpoint string)
	RecordChproxyRows(endpoint string, count float64)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordChproxyRequest(string)         {}
func (NoopMetrics) RecordChproxyRows(string, float64)   {}

type prometheusMetrics struct {
	requestsTotal *prometheus.CounterVec
	rowsTotal     *prometheus.CounterVec
}

func (m *prometheusMetrics) RecordChproxyRequest(endpoint string) {
	m.requestsTotal.WithLabelValues(endpoint).Inc()
}

func (m *prometheusMetrics) RecordChproxyRows(endpoint string, count float64) {
	m.rowsTotal.WithLabelValues(endpoint).Add(count)
}

// NewPrometheusMetrics creates and registers chproxy metrics on the given registry.
func NewPrometheusMetrics(reg prometheus.Registerer, cl prometheus.Labels) Metrics {
	m := &prometheusMetrics{
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "chproxy",
			Name:        "requests_total",
			Help:        "Total number of ClickHouse proxy requests processed.",
			ConstLabels: cl,
		}, []string{"endpoint"}),
		rowsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "chproxy",
			Name:        "rows_total",
			Help:        "Total number of rows/events processed by ClickHouse proxy.",
			ConstLabels: cl,
		}, []string{"endpoint"}),
	}

	reg.MustRegister(m.requestsTotal)
	reg.MustRegister(m.rowsTotal)

	return m
}
