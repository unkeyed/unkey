package gcra

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/gcra/db"
	"github.com/unkeyed/unkey/pkg/gcra/metrics"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
)

// readPeers returns the cached combined state of every row of name except the
// row of writerID, which the entry holds in memory.
func (l *Limiter) readPeers(ctx context.Context, name, writerID string) (State, error) {
	value, _, err := l.peerCache.SWR(ctx, writerID+"/"+name, func(ctx context.Context) (State, error) {
		start := time.Now()
		ctx, cancel := context.WithTimeout(ctx, peerReadTimeout)
		defer cancel()
		ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "gcra.read", Route: ""})
		state, err := l.circuitBreakerRead.Do(ctx, func(ctx context.Context) (State, error) {
			row, err := db.Query.ReadPeers(ctx, l.database.RO(), db.ReadPeersParams{Name: name, WriterID: writerID})
			return State{SpentNs: row.SpentNs, FreshStartNs: row.FreshStartNs}, err
		})
		metrics.ReadDurationSeconds.Observe(time.Since(start).Seconds())
		if err != nil {
			metrics.ReadsTotal.WithLabelValues("error").Inc()
			return State{}, fault.Wrap(err, fault.Internal("read gcra peer rows"))
		}
		metrics.ReadsTotal.WithLabelValues("ok").Inc()
		return state, nil
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})
	return value, err
}
