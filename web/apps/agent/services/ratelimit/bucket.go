package ratelimit

import (
	"fmt"
	"sync"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
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
	sync.RWMutex
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

// getBucket returns a bucket for the given key and will create one if it does not exist.
// It returns the bucket and a boolean indicating if the bucket existed before.
func (s *service) getBucket(key bucketKey) (*bucket, bool) {
	s.bucketsMu.RLock()
	b, ok := s.buckets[key.toString()]
	s.bucketsMu.RUnlock()
	if !ok {
		b = &bucket{
			limit:    key.limit,
			duration: key.duration,
			windows:  make(map[int64]*ratelimitv1.Window),
		}
		s.bucketsMu.Lock()
		s.buckets[key.toString()] = b
		s.bucketsMu.Unlock()
	}
	return b, ok
}

// must be called while holding a lock on the bucket
func (b *bucket) getCurrentWindow(now time.Time) *ratelimitv1.Window {
	sequence := calculateSequence(now, b.duration)

	w, ok := b.windows[sequence]
	if !ok {
		w = newWindow(sequence, now.Truncate(b.duration), b.duration)
		b.windows[sequence] = w
	}

	return w
}

// must be called while holding a lock on the bucket
func (b *bucket) getPreviousWindow(now time.Time) *ratelimitv1.Window {
	sequence := calculateSequence(now, b.duration) - 1

	w, ok := b.windows[sequence]
	if !ok {
		w = newWindow(sequence, now.Add(-b.duration).Truncate(b.duration), b.duration)
		b.windows[sequence] = w
	}

	return w
}
