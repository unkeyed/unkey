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

// Hops bucket strings. Cross-region requests should normally be 1 hop;
// anything higher indicates routing drift between regions.
const (
	hops1     = "1"
	hops2     = "2"
	hops3     = "3"
	hopsManyN = "4+"
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

	// hopsTotal counts cross-region requests bucketed by hop count. Hops=1
	// is normal (one cross-region jump); higher values mean a peer
	// frontline forwarded again — a routing-config bug.
	hopsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "hops_total",
			Help:      "Cross-region forwards by hop count.",
		},
		[]string{"src_region", "dst_region", "hops"},
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

// hopsBucket maps an integer hop count to the canonical label string used
// on hopsTotal. Keeps the label set bounded regardless of how high the hop
// count goes.
func hopsBucket(n int) string {
	switch n {
	case 1:
		return hops1
	case 2:
		return hops2
	case 3:
		return hops3
	default:
		return hopsManyN
	}
}
