package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	upstreamResponseTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "upstream_response_total",
			Help:      "Total number of upstream responses by status class.",
		},
		[]string{"status_class"},
	)

	upstreamDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "upstream_duration_seconds",
			Help:      "Backend response latency in seconds by status class, excluding sentinel overhead.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"status_class"},
	)

	// proxyAbortedTotal counts client-disconnect aborts during streaming responses.
	// httputil.ReverseProxy panics with http.ErrAbortHandler when the response body
	// copy fails after headers have been flushed (typically the client went away).
	// We swallow that sentinel locally; this counter preserves visibility.
	proxyAbortedTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "aborted_total",
			Help:      "Client-disconnect aborts during streaming.",
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
	case code >= 200:
		return "2xx"
	case code >= 100:
		return "1xx"
	default:
		return "other"
	}
}
