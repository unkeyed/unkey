// Package metrics provides Prometheus metric collectors for the global counter.
//
// Snapshots that never reach the persist stage because the queue is full are
// counted by the buffer package: unkey_buffer_state_total{name="globalcounter",state="dropped"}.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// latencyBuckets span cache hits (sub-millisecond) to blocked MySQL calls
// (up to the five second timeouts), in seconds.
var latencyBuckets = []float64{
	0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5,
}

var (
	// PersistRowsTotal counts snapshot rows by what happened to them after
	// batching and compaction.
	//
	// Possible result values are:
	// - "persisted": MySQL accepted the row.
	// - "dropped": the write failed or the write circuit breaker rejected it. The
	//   row is not retried; a later Add persists the cumulative count.
	//
	// Example usage:
	//   metrics.PersistRowsTotal.WithLabelValues("dropped").Add(float64(len(rows)))
	PersistRowsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "persist_rows_total",
			Help:      "Snapshot rows by persist result",
		},
		[]string{"result"},
	)

	// MemoryEntries tracks how many entries this process holds in local memory, one
	// per counter name and writer id.
	//
	// Several counters in one process add up.
	//
	// Example usage:
	//   metrics.MemoryEntries.Inc()
	MemoryEntries = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "memory_entries",
			Help:      "Number of entries, one per counter name and writer id, held in local memory",
		},
	)

	// AddsTotal counts Add calls by result.
	//
	// Possible result values are:
	// - "ok": the change was applied and queued for MySQL.
	// - "overflow": the count, total or revision would overflow, so nothing changed.
	// - "skipped": a ConditionalAdd chose a zero delta, so nothing changed.
	//
	// Example usage:
	//   metrics.AddsTotal.WithLabelValues("ok").Inc()
	AddsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "adds_total",
			Help:      "Add calls by result",
		},
		[]string{"result"},
	)

	// GetsTotal counts Get calls by result.
	//
	// Possible result values are:
	// - "ok": the total was returned.
	// - "error": reading other writers failed, including a rejection by the read circuit breaker.
	// - "overflow": the local and peer counts together overflow int64.
	//
	// Example usage:
	//   metrics.GetsTotal.WithLabelValues("ok").Inc()
	GetsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "gets_total",
			Help:      "Get calls by result",
		},
		[]string{"result"},
	)

	// GetDurationSeconds tracks how long Get takes as the caller sees it. Cache
	// hits are sub-millisecond, so the upper tail shows MySQL misses.
	GetDurationSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "get_duration_seconds",
			Help:      "Duration of Get calls",
			Buckets:   latencyBuckets,
		},
	)

	// PersistDurationSeconds tracks how long one batch takes to persist,
	// including retries. Failed batches are included.
	PersistDurationSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "persist_duration_seconds",
			Help:      "Duration of persisting one batch of snapshot rows",
			Buckets:   latencyBuckets,
		},
	)
)
