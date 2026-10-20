package globalcounter

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/globalcounter/metrics"
)

// GetParams identifies a globally unique name, including any tenant or window scope.
type GetParams struct {
	Name string

	// PeerMaxAge, when positive, is the oldest peer sum this read accepts. An
	// older sum is reloaded before the call returns instead of being served
	// while it refreshes. Zero keeps the cache's limits of about a second
	// fresh and five seconds stale. Set it when the caller is close to a
	// decision boundary, such as a limiter that recently denied, and can pay
	// for more replica reads. Concurrent reads of a name still share one
	// replica read.
	PeerMaxAge time.Duration
}

// Get combines the local contribution with cached peer counts. It returns zero
// for an absent name, or zero with an error on read failure or total overflow.
func (c *Counter) Get(ctx context.Context, params GetParams) (int64, error) {
	start := time.Now()
	result := "error"
	defer func() {
		metrics.GetDurationSeconds.Observe(time.Since(start).Seconds())
		metrics.GetsTotal.WithLabelValues(result).Inc()
	}()

	current := c.sequence.Load()
	peers, err := c.peerTotal(ctx, params.Name, current, params.PeerMaxAge)
	if err != nil {
		return 0, err
	}
	local, err := c.localTotal(params.Name, current)
	if err == nil {
		var total int64
		if total, err = totalOf(peers, local); err == nil {
			result = "ok"
			return total, nil
		}
	}
	result = "overflow"
	return 0, err
}

// peerTotal returns the cached sum of every row of name except this writer's
// rows of the current and previous sequence, which the writer holds in memory.
// The cache key includes current, so a rotation, which changes which writer ids
// are in memory, misses instead of serving a sum that counts them twice or not
// at all.
func (c *Counter) peerTotal(ctx context.Context, name string, current int64, maxAge time.Duration) (int64, error) {
	key := fmt.Sprintf("%d/%s", current, name)
	reason := "cache"
	var opts []cache.SWROption
	if maxAge > 0 {
		reason = "max_age"
		opts = append(opts, cache.MaxAgeFresh(maxAge))
	}
	value, _, err := c.peerCache.SWR(ctx, key, func(ctx context.Context) (int64, error) {
		ctx, cancel := context.WithTimeout(ctx, peerReadTimeout)
		defer cancel()
		value, err := c.circuitBreakerRead.Do(ctx, func(ctx context.Context) (int64, error) {
			return db.Query.Read(ctx, c.database.RO(), db.ReadParams{Name: name, CurrentWriterID: c.writerID(current), PreviousWriterID: c.writerID(current - 1)})
		})
		if err != nil {
			metrics.PeerReadsTotal.WithLabelValues(reason, "error").Inc()
			return 0, fault.Wrap(err, fault.Internal("read peer counter rows"))
		}
		metrics.PeerReadsTotal.WithLabelValues(reason, "ok").Inc()
		return value, nil
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	}, opts...)
	return value, err
}
