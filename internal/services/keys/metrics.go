package keys

import "github.com/prometheus/client_golang/prometheus"

// Metrics defines behavioral methods for observing key verification operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordVerification(keyType, code string)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordVerification(string, string) {}

type prometheusMetrics struct {
	verificationsTotal *prometheus.CounterVec
}

func (m *prometheusMetrics) RecordVerification(keyType, code string) {
	m.verificationsTotal.WithLabelValues(keyType, code).Inc()
}

func NewPrometheusMetrics(reg prometheus.Registerer, cl prometheus.Labels) Metrics {
	m := &prometheusMetrics{
		verificationsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "key", Name: "verifications_total",
			Help: "Total number of key verifications by type and outcome.", ConstLabels: cl,
		}, []string{"type", "code"}),
	}

	reg.MustRegister(m.verificationsTotal)

	return m
}
