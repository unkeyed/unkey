package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// readsTotal counts reads of peer rows by result: ok, error, row_limit,
	// or invalid_rows. Every result except ok means that a pod decides
	// without the rows of its peers.
	readsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "reads_total",
			Help:      "Reads of peer rate limit sessions by result.",
		},
		[]string{"result"},
	)

	// rowsPerRead shows how close reads get to `maxRowsPerRead`.
	rowsPerRead = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "rows_per_read",
			Help:      "Rows returned by one read of peer rate limit sessions.",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 750, 900, 1000},
		},
	)

	// cleanupsTotal counts cleanups of one fingerprint by result: applied,
	// unchanged, conflict, error, or invalid_rows. A conflict means that
	// another pod or a late write changed the rows first.
	cleanupsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "cleanups_total",
			Help:      "Cleanups of one rate limit's sessions by result.",
		},
		[]string{"result"},
	)

	// rowsPerCleanupRead shows how close cleanup reads get to
	// `maxRowsPerCleanupRead`. A full read cannot delete an idle limit.
	rowsPerCleanupRead = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "rows_per_cleanup_read",
			Help:      "Rows returned by one cleanup read of rate limit sessions.",
			Buckets:   []float64{1, 5, 10, 50, 100, 500, 1000, 2500, 4000, 4500, 5000},
		},
	)
)
