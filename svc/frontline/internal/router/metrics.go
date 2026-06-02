package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// Decision label values for routingDecisionsTotal. Match the
// router.Destination enum semantically — local for same-region instance,
// remote for cross-region peer.
const (
	decisionLocal  = "local"
	decisionRemote = "remote"
)

// routingDecisionsTotal counts where each request was routed. The
// target_region label carries the local region.platform for local
// decisions and the destination region.platform for remote decisions, so
// dashboards can attribute traffic flow between regions without joining.
//
// Routing failures are not counted here — they show up in the unified
// requests_total counter via the URN-coded fault that the router returns
// to the middleware.
var routingDecisionsTotal = lazy.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "routing_decisions_total",
		Help:      "Routing decisions by type and target region.",
	},
	[]string{"decision", "target_region"},
)

// routingDecisionSeconds is the time spent picking a destination — the
// cache lookups (route, instances, policies), DB fallbacks on miss, and
// the selection logic that produces a RouteDecision. Pairs with
// routingDecisionsTotal: that counts the decisions, this times them.
//
// Distribution is bimodal: cache hits land sub-ms, DB fallbacks tens of
// ms. The bucket set spans both regimes.
var routingDecisionSeconds = lazy.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "routing_decision_seconds",
		Help:      "Time spent picking a destination for a request.",
		Buckets:   []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	},
)
