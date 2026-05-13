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
	//   - workspace_id: the workspace the request was attributed to.
	//   - source: "local" when the decision used only in-memory counters;
	//     "origin" when a cold-window or strict-mode Redis fetch was required.
	//   - outcome: "passed" (request allowed) or "denied" (limit exceeded).
	//
	// Example usage:
	//   metrics.RatelimitDecision.WithLabelValues(workspaceID, "local", "passed").Inc()
	//   metrics.RatelimitDecision.WithLabelValues(workspaceID, "origin", "denied").Inc()
	RatelimitDecision = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "decisions_total",
			Help:      "Total number of rate-limit decisions, labeled by workspace_id, source (local|origin) and outcome (passed|denied).",
		},
		[]string{"workspace_id", "source", "outcome"},
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
	// mode for a (workspace, namespace, identifier, duration) tuple. Until the
	// deadline passes, subsequent requests on that tuple force a synchronous
	// origin fetch to converge local state. Spikes correlate with sustained
	// denial pressure and an increase in origin-fetch latency on the hot path.
	//
	// Labels:
	//   - workspace_id: the workspace whose tuple entered strict mode.
	//
	// Example usage:
	//   metrics.RatelimitStrictModeActivations.WithLabelValues(workspaceID).Inc()
	RatelimitStrictModeActivations = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "strict_mode_activations_total",
			Help:      "Total number of denials that raised the strict-mode deadline for a rate-limit tuple, labeled by workspace_id.",
		},
		[]string{"workspace_id"},
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

// Cross-region propagation: tracks the blocklist write/sync path used to
// share denials across regions through MySQL. Gives observability into
// whether the propagation channel is healthy and how much it is doing.
var (
	// RatelimitBlocklistWritesTotal counts denial events successfully written
	// to ratelimit_blocklist by the batched flush. Reflects unique strict-mode
	// transitions across the fleet, not request volume; sustained denial
	// streams from the same identifier are deduped to one write per window.
	//
	// Example usage:
	//   metrics.RatelimitBlocklistWritesTotal.Add(float64(len(batch)))
	RatelimitBlocklistWritesTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "blocklist_writes_total",
			Help:      "Total number of denial events written to ratelimit_blocklist for cross-region propagation.",
		},
	)

	// RatelimitBlocklistWriteErrors counts batch flushes that failed (MySQL
	// error or circuit-breaker trip). The events were dropped; they will
	// re-emit on the next strict-mode transition. Sustained non-zero values
	// indicate the propagation channel is impaired.
	//
	// Example usage:
	//   metrics.RatelimitBlocklistWriteErrors.Inc()
	RatelimitBlocklistWriteErrors = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "blocklist_write_errors_total",
			Help:      "Total number of ratelimit_blocklist batch flushes that failed.",
		},
	)

	// RatelimitBlocklistSyncRowsApplied counts rows pulled from
	// ratelimit_blocklist and applied to local counter state on each sync
	// tick. Each row may correspond to a denial in a remote region; the
	// inflate operation is idempotent across ticks.
	//
	// Example usage:
	//   metrics.RatelimitBlocklistSyncRowsApplied.Add(float64(len(rows)))
	RatelimitBlocklistSyncRowsApplied = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "blocklist_sync_rows_applied_total",
			Help:      "Total number of ratelimit_blocklist rows applied to local counter state.",
		},
	)

	// RatelimitBlocklistSyncErrors counts sync ticks that failed to read
	// ratelimit_blocklist. Local state remains as it was at the previous
	// successful sync.
	//
	// Example usage:
	//   metrics.RatelimitBlocklistSyncErrors.Inc()
	RatelimitBlocklistSyncErrors = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "blocklist_sync_errors_total",
			Help:      "Total number of ratelimit_blocklist sync ticks that returned an error.",
		},
	)

	// RatelimitBlocklistEntriesCreated counts counter entries that the sync
	// loop inserted because no local traffic had touched that key yet. This
	// is separate from RatelimitWindowsCreated so the traffic-driven
	// cardinality signal is not polluted by cross-region propagation.
	//
	// Example usage:
	//   metrics.RatelimitBlocklistEntriesCreated.Inc()
	RatelimitBlocklistEntriesCreated = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "blocklist_entries_created_total",
			Help:      "Total number of counter entries created by the cross-region blocklist sync loop.",
		},
	)

	// RatelimitBlocklistRowsLastPoll is the row count returned by the most
	// recent BlocklistListActive query. Set on every successful sync tick.
	// Multiplied by node count and sync frequency, this is the dominant read
	// load the propagation channel puts on MySQL — watch it to estimate
	// fleet-wide DB pressure as the active blocklist grows.
	//
	// Example usage:
	//   metrics.RatelimitBlocklistRowsLastPoll.Set(float64(len(rows)))
	RatelimitBlocklistRowsLastPoll = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "ratelimit",
			Name:      "blocklist_rows_last_poll",
			Help:      "Number of rows returned by the most recent ratelimit_blocklist sync query.",
		},
	)
)
