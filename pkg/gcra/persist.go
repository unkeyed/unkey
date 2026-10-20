package gcra

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/gcra/db"
	"github.com/unkeyed/unkey/pkg/gcra/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
)

// snapshot is one writer's full row at some point in time.
type snapshot struct {
	name        string
	writerID    string
	state       State
	expiresAtNs int64
}

func snapshotOf(name string, e *entry) snapshot {
	return snapshot{name: name, writerID: e.writerID, state: e.state, expiresAtNs: e.expiresAtNs}
}

// flush is the batch callback. It merges the snapshots of each row by taking
// the larger of every value, which is what MySQL keeps anyway, and marks what
// MySQL accepted as persisted.
func (l *Limiter) flush(ctx context.Context, snapshots []snapshot) {
	rows := mergePerRow(snapshots)
	if l.persist(ctx, rows) == nil {
		l.acknowledge(rows)
	}
}

// persist writes rows to the primary. A batch that still fails after retries
// is dropped; the maintenance pass queues dirty entries again.
func (l *Limiter) persist(ctx context.Context, rows []snapshot) error {
	start := time.Now()
	defer func() { metrics.PersistDurationSeconds.Observe(time.Since(start).Seconds()) }()
	ctx, cancel := context.WithTimeout(ctx, persistTimeout)
	defer cancel()
	ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "gcra.persist", Route: ""})

	params := array.Map(rows, func(s snapshot) db.PersistParams {
		return db.PersistParams{
			Name:         s.name,
			WriterID:     s.writerID,
			SpentNs:      s.state.SpentNs,
			FreshStartNs: s.state.FreshStartNs,
			ExpiresAtNs:  s.expiresAtNs,
		}
	})
	_, err := l.circuitBreakerWrite.Do(ctx, func(ctx context.Context) (int64, error) {
		return mysql.WithRetryContext(ctx, func() (int64, error) {
			return int64(len(params)), db.BulkQuery.Persist(ctx, l.database.RW(), params)
		})
	})
	if err != nil {
		metrics.PersistRowsTotal.WithLabelValues("dropped").Add(float64(len(rows)))
		logger.Error("dropping gcra rows after persist failed", "error", err.Error(), "batch_size", len(rows))
		return err
	}
	metrics.PersistRowsTotal.WithLabelValues("persisted").Add(float64(len(rows)))
	return nil
}

// acknowledge records what MySQL accepted. An entry that was forgotten or
// replaced by a new writer id in the meantime is left alone.
func (l *Limiter) acknowledge(rows []snapshot) {
	for _, row := range rows {
		s := l.shard(row.name)
		s.mu.Lock()
		if e, ok := s.entries[row.name]; ok && e.writerID == row.writerID {
			e.persisted.SpentNs = max(e.persisted.SpentNs, row.state.SpentNs)
			e.persisted.FreshStartNs = max(e.persisted.FreshStartNs, row.state.FreshStartNs)
		}
		s.mu.Unlock()
	}
}

// mergePerRow keeps one snapshot per name and writer id, with the largest of
// each value. Every value only grows, so the merged row is the newest state.
func mergePerRow(snapshots []snapshot) []snapshot {
	slices.SortFunc(snapshots, func(a, b snapshot) int {
		if order := cmp.Compare(a.name, b.name); order != 0 {
			return order
		}
		return cmp.Compare(a.writerID, b.writerID)
	})
	merged := snapshots[:0]
	for _, s := range snapshots {
		last := len(merged) - 1
		if last >= 0 && merged[last].name == s.name && merged[last].writerID == s.writerID {
			merged[last].state.SpentNs = max(merged[last].state.SpentNs, s.state.SpentNs)
			merged[last].state.FreshStartNs = max(merged[last].state.FreshStartNs, s.state.FreshStartNs)
			merged[last].expiresAtNs = max(merged[last].expiresAtNs, s.expiresAtNs)
			continue
		}
		merged = append(merged, s)
	}
	return merged
}

// dirtySnapshots returns a snapshot of every entry MySQL has not caught up with.
func (l *Limiter) dirtySnapshots() []snapshot {
	var pending []snapshot
	for _, s := range l.shards {
		s.mu.Lock()
		for name, e := range s.entries {
			if e.dirty() {
				pending = append(pending, snapshotOf(name, e))
			}
		}
		s.mu.Unlock()
	}
	return pending
}
