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
	// upstreamSeconds is the wall-clock duration of the cross-region
	// peer-frontline hop — request send to response complete (or error).
	//
	// Recorded only for destination=frontline (the peer hop we own end to
	// end). destination=instance is intentionally not recorded: it would
	// be dominated by customer handler time, which we do not control and
	// which would pollute our latency SLIs.
	//
	// Buckets cover the full range up to the 300s function timeout — the
	// peer may be serving an intentionally long upstream call.
	upstreamSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "upstream_seconds",
			Help:      "Cross-region peer-frontline call duration.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		},
	)

	// hopsTotal counts cross-region forwards from this instance, one per
	// ForwardToRegion call. depth is the inbound forward-chain depth:
	// depth="1" is the healthy case (we forward a request that arrived
	// directly, no prior frontline hop). depth="2_plus" means the inbound
	// request already carried a hop count — a forward-of-a-forward, almost
	// always a routing-config loop.
	//
	// Note: depth depends on X-Unkey-Frontline-Hops surviving on the
	// inbound request. WithReservedHeaderStrip drops X-Unkey-* at the HTTPS
	// edge, so today every forward records depth="1" and the total is still
	// the useful signal (volume of cross-region forwards). depth becomes
	// meaningful only if the hop header is exempted from the strip.
	//
	// Bucketed into two values instead of a histogram because the
	// distribution is bimodal and three buckets in a histogram form
	// would just be a counter with extra cost.
	hopsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "hops_total",
			Help:      "Cross-region forwards by hop depth (\"1\" = healthy single hop, \"2_plus\" = anomaly).",
		},
		[]string{"src_region", "dst_region", "depth"},
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

	// upstreamTTFBSeconds is the time from request send to first
	// response byte on the cross-region peer-frontline hop. Useful for
	// streaming workloads where upstreamSeconds reflects total stream
	// duration; TTFB isolates "did the peer start responding quickly?"
	//
	// Recorded only for destination=frontline, same reason as
	// upstreamSeconds — instance TTFB is dominated by customer handler
	// startup and is not a frontline SLI.
	upstreamTTFBSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "upstream_ttfb_seconds",
			Help:      "Time to first response byte on the cross-region peer-frontline hop.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
	)
)
