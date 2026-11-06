package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

// Outcome labels for upstreamDialsTotal.
const (
	dialOutcomeSuccess = "success"
	dialOutcomeError   = "error"
)

// Outcome labels for upgradeSparesTotal.
const (
	spareOutcomeHit     = "hit"     // a spare carried the upgrade
	spareOutcomeMiss    = "miss"    // no spare was ready, so the upgrade dialed
	spareOutcomeStale   = "stale"   // a spare failed, so the upgrade was retried on a fresh dial
	spareOutcomeReused  = "reused"  // an idle connection from a previous non-upgrade response was reused
	spareOutcomeSkipped = "skipped" // the upgrade is not a GET without a body, so it never uses spares
)

var (
	// upstreamSeconds is the wall-clock duration of the upstream call —
	// from request send to response complete (or error). Customer-pod
	// time; not a platform SLO. Surfaced for dashboard and per-tenant
	// debugging, not for platform alerts.
	//
	// Buckets cover the full range up to the 300s function timeout —
	// these requests can be intentionally long.
	upstreamSeconds = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "upstream_seconds",
			Help:      "Upstream call duration.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		},
		vec.Values("destination", "instance", "frontline"),
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
	hopsHistogram = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "hops",
			Help:      "Cross-region hop count distribution by source and destination region.",
			Buckets:   []float64{1, 2, 3},
		},
		vec.Dynamic("src_region"),
		vec.Dynamic("dst_region"),
	)

	// upstreamDialsTotal counts new TCP dials issued by the upstream
	// transport. Healthy operation reuses pooled connections; a high dial
	// rate per destination signals pool churn — a leading indicator of
	// upstream-saturation outages before errors fire.
	upstreamDialsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "upstream_dials_total",
			Help:      "TCP dials issued for upstream connections.",
		},
		vec.Values("destination", "instance", "frontline"),
		vec.Values("outcome", "success", "error"),
	)

	// upgradeSparesTotal counts cross-region protocol upgrades by how they got
	// a peer connection. A miss, stale or skipped upgrade pays TCP and TLS
	// round trips to the peer region before the upgrade.
	upgradeSparesTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "region_upgrade_spares_total",
			Help:      "Cross-region upgrades by spare connection outcome.",
		},
		[]string{"outcome"},
	)
)
