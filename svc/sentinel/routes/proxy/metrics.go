package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// upstreamResponseTotal counts upstream responses by status class.
	upstreamResponseTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_upstream_response_total",
			Help: "Total number of upstream responses by status class.",
		},
		[]string{"status_class"},
	)

	// upstreamDuration tracks backend (customer pod) response latency,
	// excluding sentinel overhead.
	upstreamDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sentinel_upstream_duration_seconds",
			Help:    "Backend response latency in seconds, excluding sentinel overhead.",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func upstreamStatusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
