package globalcounter

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/globalcounter/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
)

// snapshot is one entry's state, queued for persisting.
type snapshot struct {
	key entryKey
	row db.PersistParams
}

// persist is the batch callback. It keeps the newest snapshot of each row, writes
// them and retries transient failures through the write circuit breaker. A
// failed write drops its rows, with a log line and a count. A later Add of the
// same name persists the cumulative count again.
func (c *Counter) persist(ctx context.Context, snapshots []snapshot) {
	snapshots = newestPerRow(snapshots)

	start := time.Now()
	defer func() { metrics.PersistDurationSeconds.Observe(time.Since(start).Seconds()) }()
	ctx, cancel := context.WithTimeout(ctx, persistTimeout)
	defer cancel()
	ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "globalcounter.persist", Route: ""})

	rows := make([]db.PersistParams, len(snapshots))
	for i, snap := range snapshots {
		rows[i] = snap.row
	}
	_, err := c.circuitBreakerWrite.Do(ctx, func(ctx context.Context) (int64, error) {
		return mysql.WithRetryContext(ctx, func() (int64, error) {
			return int64(len(rows)), (&db.BulkQueries{}).Persist(ctx, c.database.RW(), rows)
		})
	})
	if err != nil {
		metrics.PersistRowsTotal.WithLabelValues("dropped").Add(float64(len(rows)))
		logger.Error("dropping counter snapshots after persist failed", "error", err.Error(), "batch_size", len(rows))
		return
	}
	metrics.PersistRowsTotal.WithLabelValues("persisted").Add(float64(len(rows)))
}

// newestPerRow keeps the newest snapshot of each row, identified by name and
// sequence. Snapshots hold absolute counts, not deltas, so older ones of the same
// row can be discarded, while the same name under another sequence is a different
// row and must be kept. It sorts by name, sequence and descending revision so that
// CompactFunc keeps the first of each row. This works in place, without a map.
//
// Example for one writer (name, sequence, revision, count):
//
//	Buffered:  (xxx, 5, 7, 100), (xxx, 6, 1, 4), (xxx, 5, 8, 80)
//	Sorted:    (xxx, 5, 8, 80), (xxx, 5, 7, 100), (xxx, 6, 1, 4)
//	Compacted: (xxx, 5, 8, 80), (xxx, 6, 1, 4)
//
// Revision 8 preserves the deduction to 80, and sequence 6 keeps its own 4.
func newestPerRow(snapshots []snapshot) []snapshot {
	slices.SortFunc(snapshots, func(a, b snapshot) int {
		if order := cmp.Compare(a.key.name, b.key.name); order != 0 {
			return order
		}
		if order := cmp.Compare(a.key.sequence, b.key.sequence); order != 0 {
			return order
		}
		return cmp.Compare(b.row.Revision, a.row.Revision)
	})
	return slices.CompactFunc(snapshots, func(a, b snapshot) bool { return a.key == b.key })
}
