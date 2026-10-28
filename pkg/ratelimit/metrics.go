package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

var (
	// readsTotal counts reads of peer rows by result: ok, error, saturated,
	// or invalid_rows. saturated means that `readsInFlightMax` reads were
	// running, so the read did not start. Every result except ok means that
	// a pod decides without the latest rows of its peers.
	readsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "reads_total",
			Help:      "Reads of rate limit rows by result.",
		},
		vec.Values("result", "ok", "error", "saturated", "invalid_rows"),
	)

	// readDurationSeconds measures reads of peer rows, including failed ones.
	readDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "read_duration_seconds",
			Help:      "Duration of one read of rate limit rows.",
			Buckets:   []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
	)

	// flushesTotal counts flushes by result: ok or error. An error leaves
	// rows uncommitted, so the next flush sends them again.
	flushesTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "flushes_total",
			Help:      "Flushes of rate limit rows by result.",
		},
		vec.Values("result", "ok", "error"),
	)

	// flushDurationSeconds measures one flush, all chunks together.
	flushDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "flush_duration_seconds",
			Help:      "Duration of one flush of rate limit rows.",
			Buckets:   []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		},
	)

	// rowsWrittenTotal counts rows sent to the store by result: ok or error.
	// Its rate is the write load that the design bounds by active limits.
	rowsWrittenTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "rows_written_total",
			Help:      "Rate limit rows written by flushes, by result.",
		},
		vec.Values("result", "ok", "error"),
	)

	// entries is the number of limits a pod holds in memory after a flush.
	entries = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "entries",
			Help:      "Rate limits held in memory by this pod.",
		},
	)

	// periodsPerRead shows how close reads get to `maxPeriodsPerRead`.
	periodsPerRead = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "periods_per_read",
			Help:      "Periods returned by one read of rate limit rows.",
			Buckets:   []float64{1, 2, 3, 4, 5, 6, 8, 12, 25, 50, 100},
		},
	)

	// cleanupPassesTotal counts cleanup passes by result: complete, bound, or
	// error. bound means the pass stopped at its bound with faded rows left,
	// so cleanup falls behind; the next pass starts within about a second.
	cleanupPassesTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "cleanup_passes_total",
			Help:      "Cleanup passes by result.",
		},
		vec.Values("result", "complete", "bound", "error"),
	)

	// rowsDeletedTotal counts the faded rows that cleanup deleted.
	rowsDeletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "rows_deleted_total",
			Help:      "Faded rate limit rows deleted by cleanup.",
		},
	)

	// clockSkewExceededTotal counts cleanup passes that found the pod clock
	// `maxClockSkew` or more away from the MySQL clock. Then cleanup may
	// delete a row while its level is still above `negligibleUnits` for the
	// pods.
	clockSkewExceededTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "gcra",
			Name:      "clock_skew_exceeded_total",
			Help:      "Cleanup passes that found the pod clock too far from the MySQL clock.",
		},
	)
)
