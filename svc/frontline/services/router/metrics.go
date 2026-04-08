package router

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics for the router service.
type Metrics struct {
	DecisionsTotal *prometheus.CounterVec
	ErrorsTotal    *prometheus.CounterVec
	Duration       prometheus.Histogram
}

// NewMetrics creates and registers all router metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		DecisionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "routing_decisions_total",
				Help:      "Total routing decisions by type and target region.",
			},
			[]string{"decision", "target_region"},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "routing_errors_total",
				Help:      "Total routing errors by reason.",
			},
			[]string{"reason"},
		),
		Duration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "routing_duration_seconds",
				Help:      "Time spent making routing decisions.",
				Buckets:   []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
		),
	}

	reg.MustRegister(m.DecisionsTotal, m.ErrorsTotal, m.Duration)

	return m
}
