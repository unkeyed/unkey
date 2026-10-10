// Package metrics declares the Prometheus metrics exported by undns. Metric
// names and labels are referenced by the operator docs and dashboards.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// QueriesTotal counts DNS responses. Reason is a bounded label such as
	// answer, no_ready_endpoints, or upstream_timeout; it never contains names.
	//
	// Labels:
	//   - "path": "private", "public", or "invalid"
	//   - "reason": why the response has its response code
	//   - "rcode": the DNS response code, such as "NOERROR" or "SERVFAIL"
	//   - "transport": "udp" or "tcp"
	QueriesTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "dns",
			Name:      "queries_total",
			Help:      "DNS responses by zone path, the reason for the response code, response code, and transport.",
		},
		[]string{"path", "reason", "rcode", "transport"},
	)

	// QueryDurationSeconds observes query latency. Public queries include
	// upstream forwarding.
	//
	// Labels:
	//   - "path": "private", "public", or "invalid"
	QueryDurationSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "dns",
			Name:      "query_duration_seconds",
			Help:      "DNS query latency by zone path, including upstream forwarding for public names.",
			Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
		[]string{"path"},
	)

	// ShedTotal counts queries answered with SERVFAIL because a concurrency
	// limit was full.
	//
	// Labels:
	//   - "limit": "queries", "forwards", "forwards_per_workspace", or "forwards_unidentified"
	ShedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "dns",
			Name:      "shed_total",
			Help:      "Queries answered with SERVFAIL because a concurrency limit was full.",
		},
		[]string{"limit"},
	)

	// DiscoveryReady is 1 while every private discovery watch is synchronized
	// and fresh. Discovery refreshes it once per second.
	DiscoveryReady = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "dns",
			Name:      "discovery_ready",
			Help:      "Whether private discovery is synchronized and fresh.",
		},
	)

	// DiscoveryWatchHealthy is 1 while one discovery watch is synchronized and
	// renewed within twice watch_timeout. Discovery refreshes it once per second.
	//
	// Labels:
	//   - "resource": "pods", "connections", "services", "endpointslices", or "topology"
	DiscoveryWatchHealthy = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "dns",
			Name:      "discovery_watch_healthy",
			Help:      "Whether a discovery watch is synchronized and renewed within twice watch_timeout.",
		},
		[]string{"resource"},
	)

	// UpstreamHealthy is 1 when the most recent public upstream exchange
	// returned a valid answer other than SERVFAIL.
	UpstreamHealthy = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "dns",
			Name:      "upstream_healthy",
			Help:      "Whether the most recent public upstream exchange succeeded.",
		},
	)

	// Connections reports published connection ConfigMaps by the answer a
	// query for them would get, as of the last connection status pass. Every
	// kind and state is reported, including zeros.
	//
	// Labels:
	//   - "kind": "connection" or "replica"
	//   - "state": "active" or a failure reason such as "service_missing"
	Connections = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "dns",
			Name:      "connections",
			Help:      "Published connection ConfigMaps by kind and by the answer a query for them would get, from the last status pass.",
		},
		[]string{"kind", "state"},
	)
)
