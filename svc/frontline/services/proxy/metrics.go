package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// proxyForwardTotal tracks proxy forwarding attempts by target type and outcome.
	// "error" label values: "none", "timeout", "conn_refused", "conn_reset",
	// "dns_failure", "client_canceled", "sentinel_5xx", "other".
	proxyForwardTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline_proxy",
			Name:      "forward_total",
			Help:      "Total proxy forward attempts by target and error type.",
		},
		[]string{"target", "error"},
	)

	// proxyBackendDuration tracks the time from when the proxy sends the request
	// to the backend until it gets a response (or error). This isolates backend
	// latency from frontline's own routing overhead.
	proxyBackendDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline_proxy",
			Name:      "backend_duration_seconds",
			Help:      "Backend response time by target type (sentinel or region).",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"target"},
	)

	// proxyHopsTotal tracks cross-region hop counts on incoming requests.
	// Values > 1 indicate multi-hop routing which should be rare.
	// Sustained high values suggest routing table issues.
	proxyHopsTotal = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline_proxy",
			Name:      "hops",
			Help:      "Distribution of frontline hop counts on cross-region requests.",
			Buckets:   []float64{0, 1, 2, 3},
		},
	)

	// proxyBackendResponseTotal tracks HTTP status codes returned by backends.
	// The "source" label distinguishes WHO produced the response:
	//   "sentinel" = sentinel itself errored (X-Unkey-Error-Source: sentinel)
	//   "upstream" = customer pod response proxied through sentinel
	//
	// Example alerts:
	//   rate(backend_response_total{source="sentinel",status_class="5xx"}[5m]) > 0 → sentinels crashing
	//   rate(backend_response_total{source="upstream",status_class="5xx"}[5m]) > X  → customer pods unhealthy
	proxyBackendResponseTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline_proxy",
			Name:      "backend_response_total",
			Help:      "Backend HTTP response status classes by target and error source.",
		},
		[]string{"target", "source", "status_class"},
	)
)
