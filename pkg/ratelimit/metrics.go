package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

var (
	// readsTotal counts reads of peer rows by result: ok, error, saturated,
	// row_limit, or invalid_rows. saturated means that `readsInFlightMax`
	// reads were running, so the read did not start. Every result except ok means that a pod decides
	// without the rows of its peers.
	readsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "reads_total",
			Help:      "Reads of rate limit rows by result.",
		},
		vec.Values("result", "ok", "error", "saturated", "row_limit", "invalid_rows"),
	)

	// bucketsPerRead shows how close reads get to `maxBucketsPerRead`.
	bucketsPerRead = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "buckets_per_read",
			Help:      "Buckets returned by one read of rate limit rows.",
			Buckets:   []float64{1, 2, 4, 8, 12, 16, 25, 50, 100, 250, 500, 1000},
		},
	)

	// cleanupsTotal counts cleanups of one fingerprint by result: applied,
	// unchanged, conflict, error, invalid_rows, or row_limit. A conflict
	// means that another pod changed the rows first. row_limit means that one
	// bucket has more rows than a cleanup read returns, so it cannot merge.
	cleanupsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "cleanups_total",
			Help:      "Cleanups of one rate limit's rows by result.",
		},
		vec.Values("result", "applied", "unchanged", "conflict", "error", "invalid_rows", "row_limit"),
	)

	// cleanupPassesTotal counts cleanup passes by result: complete, bound, or
	// error. bound means the pass stopped at one of its bounds with due rows
	// left, so cleanup falls behind; the next pass starts within about a
	// second.
	cleanupPassesTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "cleanup_passes_total",
			Help:      "Cleanup passes by result.",
		},
		vec.Values("result", "complete", "bound", "error"),
	)

	// rowsPerCleanupRead shows how close cleanup reads get to
	// `maxRowsPerCleanupRead`. A full read leaves its newest bucket for the
	// next cleanup.
	rowsPerCleanupRead = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "rows_per_cleanup_read",
			Help:      "Rows returned by one cleanup read of rate limit rows.",
			Buckets:   []float64{1, 5, 10, 50, 100, 500, 1000, 2500, 4000, 4500, 5000},
		},
	)

	// clockSkewExceededTotal counts cleanup passes that found the pod clock
	// `maxClockSkew` or more away from the MySQL clock. Then MySQL may
	// refuse writes that the pod still sends, or cleanup may merge rows that
	// the pod still counts.
	clockSkewExceededTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "clock_skew_exceeded_total",
			Help:      "Cleanup passes that found the pod clock too far from the MySQL clock.",
		},
	)
)
