package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// proxyForwardTotal tracks proxy forwarding attempts by destination and outcome.
	//
	// Labels:
	//   destination: "sentinel" (local h2c forward) or "region" (cross-region HTTPS forward)
	//   error: "none", "timeout", "conn_refused", "conn_reset",
	//          "dns_failure", "client_canceled", "backend_5xx", "other"
	proxyForwardTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "forward_total",
			Help:      "Total proxy forward attempts by destination and error type.",
		},
		[]string{"destination", "error"},
	)

	// proxyBackendDuration tracks the time from when the proxy sends the request
	// to the backend until it gets a response (or error). This isolates backend
	// latency from frontline's own routing overhead.
	//
	// Labels:
	//   destination: "sentinel" (local h2c forward) or "region" (cross-region HTTPS forward)
	proxyBackendDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "backend_duration_seconds",
			Help:      "Backend response time by destination type.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"destination"},
	)

	// proxyHopsTotal tracks cross-region hop counts on incoming requests.
	// Values > 1 indicate multi-hop routing which should be rare.
	// Sustained high values suggest routing table issues.
	proxyHopsTotal = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "hops",
			Help:      "Distribution of frontline hop counts on cross-region requests.",
			Buckets:   []float64{0, 1, 2, 3},
		},
	)

	// proxyBackendResponseTotal tracks HTTP status codes returned by backends.
	//
	// Labels:
	//   destination: "sentinel" (local h2c forward) or "region" (cross-region HTTPS forward)
	//   source: "sentinel" (sentinel itself errored) or "upstream" (customer pod response)
	//   status_class: "2xx", "3xx", "4xx", "5xx"
	proxyBackendResponseTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "backend_response_total",
			Help:      "Backend HTTP response status classes by destination and error source.",
		},
		[]string{"destination", "source", "status_class"},
	)
)
