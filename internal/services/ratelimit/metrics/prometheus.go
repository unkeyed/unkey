// Package metrics provides Prometheus collectors for the rate limit service.
//
// Metrics are organized around three concerns:
//
//   - Counter lifecycle: how many sliding-window counters exist, how often
//     they're created and evicted. Tracks memory pressure and cardinality.
//   - Decision outcomes: whether each request passed or was denied, and
//     whether the decision was made from local state or required an
//     origin fetch.
//   - Origin health: latency and error rates for Redis operations that
//     the ratelimit service depends on for cross-node convergence.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// latencyBuckets are standard histogram buckets for latency metrics in seconds.
// Range covers sub-millisecond local decisions through multi-second origin
// failures where the circuit breaker should have tripped.
var latencyBuckets = []float64{
	0.001, // 1ms
	0.002, // 2ms
	0.005, // 5ms
	0.01,  // 10ms
	0.02,  // 20ms
	0.05,  // 50ms
	0.1,   // 100ms
	0.2,   // 200ms
	0.3,   // 300ms
	0.4,   // 400ms
	0.5,   // 500ms
	0.75,  // 750ms
	1.0,   // 1s
	2.0,   // 2s
	3.0,   // 3s
	5.0,   // 5s
	10.0,  // 10s
}

// Counter lifecycle: tracks the in-memory sync.Map entries that hold sliding
// window state. Each entry is one (name, identifier, duration, sequence) tuple.
var (
	// RatelimitWindows is the number of sliding-window counters currently held
	// in memory. High cardinality drives memory usage; the janitor evicts
	// counters older than 3× their window duration.
	//
	// Example usage:
	//   metrics.RatelimitWindows.Set(float64(activeCounters))
	RatelimitWindows = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "windows",
			Help:      "Current number of sliding-window counters in memory.",
		},
	)

	// RatelimitWindowsCreated counts how many sliding-window counters have
	// been created. Divided by RatelimitWindowsEvicted, this tells you whether
	// cardinality is steady or growing.
	//
	// Example usage:
	//   metrics.RatelimitWindowsCreated.Inc()
	RatelimitWindowsCreated = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "windows_created_total",
			Help:      "Total number of sliding-window counters created.",
		},
	)

	// RatelimitWindowsEvicted counts how many sliding-window counters the
	// janitor has removed. Counters are evicted when their window ends more
	// than 3× the window duration ago.
	//
	// Example usage:
	//   metrics.RatelimitWindowsEvicted.Inc()
	RatelimitWindowsEvicted = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "windows_evicted_total",
			Help:      "Total number of sliding-window counters evicted by the janitor.",
		},
	)
)

// Decision outcomes: what the service returned and how it got there.
var (
	// RatelimitDecision counts rate-limit decisions.
	//
	// Labels:
	//   - source: "local" when the decision used only in-memory counters;
	//     "origin" when a cold-window or strict-mode Redis fetch was required.
	//   - outcome: "passed" (request allowed) or "denied" (limit exceeded).
	//
	// Example usage:
	//   metrics.RatelimitDecision.WithLabelValues("local", "passed").Inc()
	//   metrics.RatelimitDecision.WithLabelValues("origin", "denied").Inc()
	RatelimitDecision = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "decisions_total",
			Help:      "Total number of rate-limit decisions, labeled by source (local|origin) and outcome (passed|denied).",
		},
		[]string{"source", "outcome"},
	)

	// RatelimitCASExhausted counts Ratelimit() calls that exhausted the CAS
	// retry budget and failed closed. This should always be zero in practice;
	// any non-zero value indicates pathological contention that warrants
	// investigation.
	//
	// Example usage:
	//   metrics.RatelimitCASExhausted.Inc()
	RatelimitCASExhausted = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "cas_exhausted_total",
			Help:      "Total number of Ratelimit() calls that exhausted CAS retries and failed closed.",
		},
	)

	// RatelimitStrictModeActivations counts how often a denial triggered strict
	// mode for a (name, identifier, duration) tuple. Until the deadline passes,
	// subsequent requests on that tuple force a synchronous origin fetch to
	// converge local state. Spikes correlate with sustained denial pressure
	// and an increase in origin-fetch latency on the hot path.
	//
	// Example usage:
	//   metrics.RatelimitStrictModeActivations.Inc()
	RatelimitStrictModeActivations = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "strict_mode_activations_total",
			Help:      "Total number of denials that raised the strict-mode deadline for a rate-limit tuple.",
		},
	)
)

// Origin (Redis) health: latency and error rates for the operations that
// keep nodes eventually consistent.
//
// Both metrics share an "op" label:
//
//   - op="fetch" — GET on the request path (cold windows and strict mode).
//     Latency here directly affects request p99.
//   - op="sync"  — INCR from the replay workers. Latency here predicts how
//     quickly local counters across nodes converge.
//
// Total operations per op are available as the histogram's implicit _count
// series, e.g. ratelimit_origin_latency_seconds_count{op="fetch"}.
var (
	// RatelimitOriginLatency observes the wall-clock latency of origin
	// operations.
	//
	// Labels:
	//   - op: "fetch" for hot-path GET calls (cold window / strict mode);
	//     "sync" for replay-worker INCR calls.
	//
	// Example usage:
	//   metrics.RatelimitOriginLatency.WithLabelValues("fetch").Observe(seconds)
	//   metrics.RatelimitOriginLatency.WithLabelValues("sync").Observe(seconds)
	RatelimitOriginLatency = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "origin_latency_seconds",
			Help:      "Latency of origin operations in seconds, labeled by op (fetch|sync).",
			Buckets:   latencyBuckets,
		},
		[]string{"op"},
	)

	// RatelimitOriginErrors counts origin operations that returned an error
	// (including circuit-breaker trips and hot-path timeouts).
	//
	// Labels:
	//   - op: "fetch" for hot-path GET calls (cold window / strict mode);
	//     "sync" for replay-worker INCR calls.
	//   - reason: "timeout" when the per-call deadline elapsed (e.g. the
	//     150ms hot-path budget on fetch); "other" for any other error
	//     surface (Redis returned an error, circuit breaker open, etc.).
	//     Sustained timeouts on op="fetch" are alert-worthy: they mean
	//     the rate-limit decision is falling back to local state because
	//     Redis is slow, which loosens cross-node convergence.
	//
	// Example usage:
	//   metrics.RatelimitOriginErrors.WithLabelValues("fetch", "timeout").Inc()
	//   metrics.RatelimitOriginErrors.WithLabelValues("sync", "other").Inc()
	RatelimitOriginErrors = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "origin_errors_total",
			Help:      "Total number of origin operations that returned an error, labeled by op (fetch|sync) and reason (timeout|other).",
		},
		[]string{"op", "reason"},
	)
)
