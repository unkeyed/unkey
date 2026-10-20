// Package metrics provides Prometheus metric collectors for the GCRA limiter.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// latencyBuckets span replica reads and primary writes up to their five second
// timeouts, in seconds.
var latencyBuckets = []float64{
	0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5,
}

var (
	// DecisionsTotal counts decisions per checked name.
	//
	// Possible outcome values are "passed" and "denied". A name of a multi-name
	// call that passed on its own counts as passed even if another name denied
	// the call.
	//
	// Example usage:
	//   metrics.DecisionsTotal.WithLabelValues("passed").Inc()
	DecisionsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "decisions_total",
			Help:      "GCRA decisions per checked name by outcome (passed|denied)",
		},
		[]string{"outcome"},
	)

	// ClampsTotal counts committed decisions that raised the writer's fresh
	// start because the name had been idle, so its idle time was forgotten.
	ClampsTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "clamps_total",
			Help:      "Committed decisions that raised the writer's fresh start after idle time",
		},
	)

	// PersistRowsTotal counts rows by persist result. Each flush writes one
	// row per name and writer.
	//
	// Possible result values are:
	// - "persisted": MySQL accepted the row.
	// - "dropped": the write failed or the write circuit breaker rejected it. The
	//   entries stay dirty and the next flush writes them again.
	//
	// Example usage:
	//   metrics.PersistRowsTotal.WithLabelValues("persisted").Add(float64(len(rows)))
	PersistRowsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "persist_rows_total",
			Help:      "GCRA rows by persist result (persisted|dropped)",
		},
		[]string{"result"},
	)

	// PersistDurationSeconds tracks how long one batch takes to persist,
	// including retries. Failed batches are included.
	PersistDurationSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "persist_duration_seconds",
			Help:      "Duration of persisting one batch of GCRA rows",
			Buckets:   latencyBuckets,
		},
	)

	// MemoryEntries tracks how many names this process holds in memory. Several
	// limiters in one process add up.
	MemoryEntries = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "memory_entries",
			Help:      "Number of GCRA entries held in local memory",
		},
	)

	// ReadsTotal counts peer reads that went to the replica, so cache hits are
	// not included.
	//
	// Possible result values are "ok" and "error". A rejection by the read
	// circuit breaker is an error.
	//
	// Example usage:
	//   metrics.ReadsTotal.WithLabelValues("ok").Inc()
	ReadsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "reads_total",
			Help:      "GCRA peer reads from the replica by result (ok|error)",
		},
		[]string{"result"},
	)

	// ReadDurationSeconds tracks how long a peer read from the replica takes.
	ReadDurationSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "read_duration_seconds",
			Help:      "Duration of GCRA peer reads from the replica",
			Buckets:   latencyBuckets,
		},
	)

	// CleanupNamesTotal counts names whose rows the cleanup pass deleted.
	CleanupNamesTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "cleanup_names_total",
			Help:      "Names whose rows the GCRA cleanup pass deleted",
		},
	)

	// CleanupNamesSkippedTotal counts names the cleanup pass skipped because
	// they have more rows than one cleanup transaction may lock. Such a name is
	// never deleted, so any increase needs attention.
	CleanupNamesSkippedTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "cleanup_names_skipped_total",
			Help:      "Names the GCRA cleanup pass skipped because they have too many rows",
		},
	)
)
