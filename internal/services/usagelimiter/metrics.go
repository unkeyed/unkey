package usagelimiter

import "github.com/prometheus/client_golang/prometheus"

// Metrics defines behavioral methods for observing usage limiter operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordDecision(source, outcome string)
	RecordReplayOperation(status string)
	RecordReplayLatency(seconds float64)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordDecision(string, string)  {}
func (NoopMetrics) RecordReplayOperation(string)   {}
func (NoopMetrics) RecordReplayLatency(float64)    {}

type prometheusMetrics struct {
	decisions        *prometheus.CounterVec
	replayOperations *prometheus.CounterVec
	replayLatency    prometheus.Histogram
}

func (m *prometheusMetrics) RecordDecision(source, outcome string) {
	m.decisions.WithLabelValues(source, outcome).Inc()
}
func (m *prometheusMetrics) RecordReplayOperation(status string) {
	m.replayOperations.WithLabelValues(status).Inc()
}
func (m *prometheusMetrics) RecordReplayLatency(seconds float64) {
	m.replayLatency.Observe(seconds)
}

func NewPrometheusMetrics(reg prometheus.Registerer, cl prometheus.Labels) Metrics {
	m := &prometheusMetrics{
		decisions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "usagelimiter", Name: "decisions_total",
			Help: "Usage limiter decisions by source and outcome.", ConstLabels: cl,
		}, []string{"source", "outcome"}),
		replayOperations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "usagelimiter", Name: "replay_operations_total",
			Help: "Total number of replay operations to the database.", ConstLabels: cl,
		}, []string{"status"}),
		replayLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "unkey", Subsystem: "usagelimiter", Name: "replay_latency_seconds",
			Help:        "Latency of replay operations in seconds.",
			Buckets:     []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 2.0, 3.0, 5.0, 10.0},
			ConstLabels: cl,
		}),
	}

	reg.MustRegister(m.decisions, m.replayOperations, m.replayLatency)

	return m
}
