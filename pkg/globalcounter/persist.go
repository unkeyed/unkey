package globalcounter

import (
	"cmp"
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/globalcounter/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
)

// snapshot pairs a row with the entry it came from, so a batch can be reduced
// per entry without parsing writer ids.
type snapshot struct {
	key entryKey
	row db.PersistParams
}

// flush is the batch callback. Only the newest snapshot of each row is written:
// it carries the full count, and MySQL would reject the older ones anyway.
func (c *Counter) flush(ctx context.Context, snapshots []snapshot) {
	snapshots = newestPerRow(snapshots)
	c.persist(ctx, array.Map(snapshots, func(s snapshot) db.PersistParams { return s.row }))
}

// persist writes rows to the primary. It retries only within one deadline, to
// ride out transient errors; a batch that still fails is dropped, not kept for
// later. Keeping it would grow memory for as long as the database is down, and
// is not needed: every row is a full snapshot, so the next snapshot of the same
// entry repairs what was lost.
func (c *Counter) persist(ctx context.Context, rows []db.PersistParams) {
	start := time.Now()
	defer func() { metrics.PersistDurationSeconds.Observe(time.Since(start).Seconds()) }()
	ctx, cancel := context.WithTimeout(ctx, persistTimeout)
	defer cancel()
	ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "globalcounter.persist", Route: ""})

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

// persistRow converts a local entry into the snapshot this writer persists for
// key. The row keeps the writer id of the entry's sequence however late it is
// published.
func (c *Counter) persistRow(key entryKey, e entry) snapshot {
	deleteAfter := sql.NullInt64{}
	if !e.deleteAfter.IsZero() {
		deleteAfter = sql.NullInt64{Int64: e.deleteAfter.UnixMilli(), Valid: true}
	}
	return snapshot{
		key: key,
		row: db.PersistParams{
			Name:           key.name,
			WriterID:       c.writerID(key.sequence),
			Count:          e.count,
			Revision:       e.revision,
			DeleteAfter:    deleteAfter,
			CompactDelayMs: c.compactDelayMs,
		},
	}
}
