package handler

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics for the proxy handler package.
type Metrics struct {
	UpstreamResponseTotal *prometheus.CounterVec
	UpstreamDuration      *prometheus.HistogramVec
}

// NewMetrics creates and registers all proxy handler metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		UpstreamResponseTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "upstream_response_total",
				Help:      "Total number of upstream responses by status class.",
			},
			[]string{"status_class"},
		),
		UpstreamDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "upstream_duration_seconds",
				Help:      "Backend response latency in seconds by status class, excluding sentinel overhead.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
			},
			[]string{"status_class"},
		),
	}

	reg.MustRegister(m.UpstreamResponseTotal)
	reg.MustRegister(m.UpstreamDuration)

	return m
}

func upstreamStatusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	case code >= 100:
		return "1xx"
	default:
		return "other"
	}
}
