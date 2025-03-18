package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// Generally there is one bucket per identifier.
// However if the same identifier is used with different config, such as limit
// or duration, there will be multiple buckets for the same identifier.
//
// A bucket is always uniquely identified by this triplet: identifier, limit, duration.
// See `bucketKey` for more details.
//
// A bucket reaches its lifetime when the last window has expired at least 1 * duration ago.
// In other words, we can remove a bucket when it is no longer relevant for
// ratelimit decisions.
type bucket struct {
	mu       sync.RWMutex
	limit    int64
	duration time.Duration
	// sequence -> window
	windows map[int64]*ratelimitv1.Window
}

// bucketKey returns a unique key for an identifier and duration config
// the duration is required to ensure a change in ratelimit config will not
// reuse the same bucket and mess up the sequence numbers
type bucketKey struct {
	identifier string
	limit      int64
	duration   time.Duration
}

func (b bucketKey) toString() string {
	return fmt.Sprintf("%s-%d-%d", b.identifier, b.limit, b.duration.Milliseconds())
}

// getOrCreateBucket retrieves a rate limiting bucket for the given key.
// If no bucket exists, it creates a new one.
//
// The bucket is uniquely identified by the combination of:
// - identifier: the client identifier being rate limited
// - limit: the maximum number of allowed requests
// - duration: the time window for applying the limit
//
// This function is thread-safe and can be called concurrently.
//
// Returns:
//   - *bucket: the bucket for tracking rate limit state
//   - bool: true if the bucket already existed, false if it was created
func (s *service) getOrCreateBucket(ctx context.Context, key bucketKey) (*bucket, bool) {
	ctx, span := tracing.Start(ctx, "getOrCreateBucket")
	defer span.End()

	s.bucketsMu.RLock()
	b, exists := s.buckets[key.toString()]
	s.bucketsMu.RUnlock()
	if !exists {

		b = &bucket{
			mu:       sync.RWMutex{},
			limit:    key.limit,
			duration: key.duration,
			windows:  make(map[int64]*ratelimitv1.Window),
		}
		s.bucketsMu.Lock()
		s.buckets[key.toString()] = b
		s.bucketsMu.Unlock()
	}
	span.SetAttributes(attribute.Bool("created", !exists))
	return b, exists
}

// must be called while holding a lock on the bucket
func (b *bucket) getCurrentWindow(ctx context.Context, now time.Time) *ratelimitv1.Window {
	ctx, span := tracing.Start(ctx, "getCurrentWindow")
	defer span.End()
	sequence := calculateSequence(now, b.duration)

	w, exists := b.windows[sequence]
	if !exists {
		w = newWindow(sequence, now.Truncate(b.duration), b.duration)
		b.windows[sequence] = w
	}
	span.SetAttributes(attribute.Bool("created", !exists))

	return w
}

// must be called while holding a lock on the bucket
func (b *bucket) getPreviousWindow(ctx context.Context, now time.Time) *ratelimitv1.Window {
	ctx, span := tracing.Start(ctx, "getCurrentWindow")
	defer span.End()

	sequence := calculateSequence(now, b.duration) - 1

	w, exists := b.windows[sequence]
	if !exists {
		w = newWindow(sequence, now.Add(-b.duration).Truncate(b.duration), b.duration)
		b.windows[sequence] = w
	}
	span.SetAttributes(attribute.Bool("created", !exists))

	return w
}
