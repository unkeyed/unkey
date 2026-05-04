package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// proxyForwardTotal tracks proxy forwarding attempts by destination and outcome.
	//
	// Labels:
	//   destination: "instance" (local h2c forward to deployment instance) or
	//                "region" (cross-region HTTPS forward to peer frontline)
	//   error: "none", "timeout", "conn_refused", "conn_reset",
	//          "dns_failure", "client_canceled", "backend_5xx", "other"
	proxyForwardTotal = lazy.NewCounterVec(
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
	//   destination: "instance" (local h2c forward) or "region" (cross-region HTTPS forward)
	proxyBackendDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "backend_duration_seconds",
			Help:      "Backend response time by destination type.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"destination"},
	)

	// proxyHops tracks cross-region hop counts by source and destination region.
	// Deviations from nominal hop counts per src→dst pair indicate routing shifts.
	proxyHops = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "hops",
			Help:      "Distribution of frontline hop counts by source and destination region.",
			Buckets:   []float64{0, 1, 2, 3},
		},
		[]string{"src_region", "dst_region"},
	)

	// proxyForwardErrorsTotal is a convenience counter for forward errors.
	proxyForwardErrorsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "forward_errors_total",
			Help:      "Total proxy forward errors by destination.",
		},
		[]string{"destination"},
	)

	// proxyBackendErrorsTotal counts backend 5xx responses by destination and source.
	proxyBackendErrorsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "backend_errors_total",
			Help:      "Total backend 5xx errors by destination and source.",
		},
		[]string{"destination", "source"},
	)

	// proxyAbortedTotal counts client-disconnect aborts during streaming responses.
	// httputil.ReverseProxy panics with http.ErrAbortHandler when the response body
	// copy fails after headers have been flushed (typically the client went away).
	// We swallow that sentinel value locally; this counter preserves visibility.
	proxyAbortedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "aborted_total",
			Help:      "Client-disconnect aborts during streaming by destination.",
		},
		[]string{"destination"},
	)

	// proxyBackendResponseTotal tracks HTTP status codes returned by backends.
	//
	// Labels:
	//   destination: "instance" (local h2c forward) or "region" (cross-region HTTPS forward)
	//   source: "frontline" (peer frontline returned this error) or "upstream" (customer pod response)
	//   status_class: "2xx", "3xx", "4xx", "5xx"
	proxyBackendResponseTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "backend_response_total",
			Help:      "Backend HTTP response status classes by destination and error source.",
		},
		[]string{"destination", "source", "status_class"},
	)
)
