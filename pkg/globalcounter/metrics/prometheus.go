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
	// PersistRowsTotal counts snapshot rows by what happened to them, after a batch
	// keeps only the newest snapshot of each row.
	//
	// Possible result values are:
	// - "persisted": MySQL accepted the row.
	// - "dropped": the write failed or the write circuit breaker rejected it. The
	//   eviction pass queues the entry again while its writer id is the previous
	//   one, so a later Add or that pass persists the cumulative count.
	// - "abandoned": the writer forgot an entry whose last snapshot MySQL never
	//   acknowledged. Its newest changes are missing from the totals.
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
	// per counter name and writer id. Entries leave memory through eviction, when
	// their deletion time passes or their writer id is older than the previous one.
	// A name without a deletion time therefore holds one entry for each of the
	// current and previous writer id, and steady growth beyond that means eviction
	// is failing.
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

	// PeerReadsTotal counts replica reads of other writers' rows, which happen
	// only on peer cache misses and refreshes. Comparing the strict cache with the
	// regular one shows the replica load strict callers add.
	//
	// Possible cache values are "peer" and "strict". Possible result values are
	// "ok" and "error".
	//
	// Example usage:
	//   metrics.PeerReadsTotal.WithLabelValues("strict", "ok").Inc()
	PeerReadsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "peer_reads_total",
			Help:      "Replica reads of other writers' rows by cache and result",
		},
		[]string{"cache", "result"},
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

	// CompactionsTotal counts compaction transactions.
	//
	// Possible result values are:
	// - "compacted": the transaction merged the rows it selected, or found none.
	// - "error": MySQL failed; the rows stay and a later pass retries.
	//
	// Example usage:
	//   metrics.CompactionsTotal.WithLabelValues("compacted").Inc()
	CompactionsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "compactions_total",
			Help:      "Compaction transactions by result",
		},
		[]string{"result"},
	)

	// CompactedRowsTotal counts rows that nobody wrote for a while, merged into
	// compacted rows.
	CompactedRowsTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "compacted_rows_total",
			Help:      "Rows nobody wrote for a while, merged into compacted rows",
		},
	)

	// CompactionDeferredRowsTotal counts rows that a compaction pass left alone and
	// moved to a higher compaction level, because their name had a single
	// compactable row or their sum overflows int64. The wait doubles with each
	// level, so a row that nobody writes costs a few updates in its first weeks and
	// almost none after that.
	CompactionDeferredRowsTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "globalcounter",
			Name:      "compaction_deferred_rows_total",
			Help:      "Rows a compaction pass left alone and moved to a higher compaction level",
		},
	)
)
