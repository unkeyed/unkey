package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// Destination labels distinguish where the proxy sent the request.
const (
	destinationInstance  = "instance"  // local h2c forward to a deployment pod
	destinationFrontline = "frontline" // cross-region HTTPS forward to peer frontline
)

// Outcome labels for upstreamDialsTotal.
const (
	dialOutcomeSuccess = "success"
	dialOutcomeError   = "error"
)

var (
	// upstreamSeconds is the wall-clock duration of the upstream call —
	// from request send to response complete (or error). Customer-pod
	// time; not a platform SLO. Surfaced for dashboard and per-tenant
	// debugging, not for platform alerts.
	//
	// Buckets cover the full range up to the 300s function timeout —
	// these requests can be intentionally long.
	upstreamSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "upstream_seconds",
			Help:      "Upstream call duration.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		},
		[]string{"destination"},
	)

	// hopsHistogram records cross-region hop counts. Hops=1 is the only
	// healthy value: a request arrives, we forward to one peer, that peer
	// serves it. Higher values mean the peer received a forward and
	// forwarded *again* — a routing-config bug between regions.
	//
	// Recording as a histogram (not a counter) so dashboards get average
	// hop depth via _sum/_count and alerts can use bucket boundaries to
	// distinguish "all good" (everything in le=1) from "drift" (any rate
	// past le=1).
	hopsHistogram = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "hops",
			Help:      "Cross-region hop count distribution by source and destination region.",
			Buckets:   []float64{1, 2, 3},
		},
		[]string{"src_region", "dst_region"},
	)

	// upstreamDialsTotal counts new TCP dials issued by the upstream
	// transport. Healthy operation reuses pooled connections; a high dial
	// rate per destination signals pool churn — a leading indicator of
	// upstream-saturation outages before errors fire.
	upstreamDialsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "upstream_dials_total",
			Help:      "TCP dials issued for upstream connections.",
		},
		[]string{"destination", "outcome"},
	)
)
