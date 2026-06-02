package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// RequestsTotal is the unified per-request outcome counter. One row per
// completed request, labelled by:
//
//	status_class 2xx | 3xx | 4xx | 5xx
//	code         "" on success, the fault URN string on error
//	             (e.g. err:unkey:not_found:config_not_found_for_custom_domain)
//	outcome      success | refused | frontline_fault | upstream_problem | noise
//	             (see OutcomeFor — mechanical mapping from URN)
//
// `outcome` is the low-cardinality bucket; `code` is the URN for
// drill-down. Region and environment are external labels on the
// per-region Prometheus and are NOT carried as metric labels.
var RequestsTotal = lazy.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "requests_total",
		Help:      "Total requests by status class, fault code (URN), and outcome.",
	},
	[]string{"status_class", "code", "outcome"},
)

// InflightRequests is the per-pod count of in-flight requests. Incremented
// on entry to the observability middleware and decremented on exit, so it
// covers every routed path.
//
// Saturation signal: if the gauge climbs while traffic is steady, an
// upstream got slow or frontline stalled. Pair with goroutine count for
// triage — a leak shows up in both.
var InflightRequests = lazy.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "inflight_requests",
		Help:      "Requests currently being processed.",
	},
)

// OverheadSeconds is the frontline latency SLI: total request time minus
// the upstream call (instance handler or peer-frontline hop). It is the
// time frontline itself spent on a request — auth, routing, policy
// evaluation, error rendering, header/body bookkeeping — and is the only
// latency number for which frontline is fully accountable.
//
// Labelled by outcome so SLIs can be expressed as e.g.
//
//	histogram_quantile(0.99,
//	  rate(unkey_frontline_overhead_seconds_bucket{outcome="success"}[5m]))
//
// Pre-routing failures (auth fail, no route) report overhead = total.
// Cross-region forwards subtract the peer-hop duration; instance-path
// requests subtract the customer-pod time.
//
// Sub-second buckets because anything ≥1s of *our* overhead is a bug.
var OverheadSeconds = lazy.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "overhead_seconds",
		Help:      "Frontline-only request latency (total minus upstream call), by outcome.",
		Buckets:   []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	},
	[]string{"outcome"},
)

// StatusClass returns the HTTP status class label for a status code:
// "2xx", "3xx", "4xx", or "5xx".
func StatusClass(code int) string {
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
