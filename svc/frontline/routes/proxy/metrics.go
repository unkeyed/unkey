package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// Outcome labels for localRequestRetriesTotal.
const (
	// retryOutcomeRecovered: at least one local instance dial-failed, but a
	// later candidate in the same region served the request successfully.
	// Healthy — retry did its job and the client never saw an error.
	retryOutcomeRecovered = "recovered"
	// retryOutcomeExhausted: every local instance dial-failed. The client
	// may have still received a 2xx if a peer-region fallback was available
	// and succeeded — that case is counted separately by regionFallbacksTotal.
	// Indicates either local capacity is degraded or the cached instance list
	// is staler than the actual cluster state.
	retryOutcomeExhausted = "exhausted"
)

// localRequestRetriesTotal counts requests routed to a *local* deployment
// instance where the per-instance retry loop had to advance past a dial
// failure. Requests that succeed on the first attempt are NOT counted —
// so the ratio recovered/(recovered+exhausted) directly measures retry
// effectiveness on the local-region path. Cross-region forwards (the
// DestinationRemoteRegion path) never hit the retry loop and are not
// represented here.
//
// Granularity is per-request, not per-attempt: the upstream-dial signal
// already lives in proxy.upstreamDialsTotal{outcome="error"}; this metric
// answers the higher-level question "did the client get a response from
// the local region?".
var localRequestRetriesTotal = lazy.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "local_request_retries_total",
		Help:      "Local-instance requests that hit at least one dial failure, labelled by final outcome.",
	},
	[]string{"outcome"},
)

// regionFallbacksTotal counts requests where every local instance dial-
// failed and the handler forwarded to a peer region instead. A non-zero
// rate means local capacity is degraded — local-region SLOs should fire
// off the underlying dial-failure metrics, NOT off the client-visible
// success rate (which the fallback masks).
//
// to_region is the peer region selected by the router as standby (e.g.
// "us-west-2.aws"). The "from" side is implicit in the scrape labels.
var regionFallbacksTotal = lazy.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "region_fallbacks_total",
		Help:      "Requests forwarded to a peer region after every local instance dial-failed.",
	},
	[]string{"to_region"},
)
