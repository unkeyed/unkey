package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics for the proxy service.
type Metrics struct {
	ForwardTotal         *prometheus.CounterVec
	BackendDuration      *prometheus.HistogramVec
	Hops                 *prometheus.HistogramVec
	BackendResponseTotal *prometheus.CounterVec
	ForwardErrorsTotal   *prometheus.CounterVec
	BackendErrorsTotal   *prometheus.CounterVec
}

// NewMetrics creates and registers all proxy metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ForwardTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "forward_total",
				Help:      "Total proxy forward attempts by destination and error type.",
			},
			[]string{"destination", "error"},
		),
		BackendDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "backend_duration_seconds",
				Help:      "Backend response time by destination type.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
			},
			[]string{"destination"},
		),
		Hops: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "hops",
				Help:      "Distribution of frontline hop counts by source and destination region.",
				Buckets:   []float64{0, 1, 2, 3},
			},
			[]string{"src_region", "dst_region"},
		),
		BackendResponseTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "backend_response_total",
				Help:      "Backend HTTP response status classes by destination and error source.",
			},
			[]string{"destination", "source", "status_class"},
		),
		ForwardErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "forward_errors_total",
				Help:      "Total proxy forward errors by destination.",
			},
			[]string{"destination"},
		),
		BackendErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "frontline",
				Name:      "backend_errors_total",
				Help:      "Total backend errors by destination and source.",
			},
			[]string{"destination", "source"},
		),
	}

	reg.MustRegister(
		m.ForwardTotal,
		m.BackendDuration,
		m.Hops,
		m.BackendResponseTotal,
		m.ForwardErrorsTotal,
		m.BackendErrorsTotal,
	)

	return m
}
