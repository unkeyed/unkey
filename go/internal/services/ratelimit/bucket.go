package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

// bucket maintains rate limit state for a specific identifier+limit+duration combination.
// It stores a sliding window of request counts and manages the lifecycle of these windows.
//
// Each bucket is uniquely identified by a triplet of:
//   - identifier: The rate limit subject (user ID, API key, etc)
//   - limit: Maximum requests allowed in the duration
//   - duration: Time window for the rate limit
//
// Example Usage:
//
//	b := &bucket{
//	    limit:    100,
//	    duration: time.Minute,
//	    windows:  make(map[int64]window),
//	}
//
//	b.mu.Lock()
//	window, _ := b.getCurrentWindow(time.Now())
//	b.mu.Unlock()
type bucket struct {
	// mu protects all bucket operations
	mu sync.RWMutex

	// limit is the maximum number of requests allowed per duration
	limit int64

	// duration is the time window for this rate limit
	duration time.Duration

	// windows maps sequence numbers to time windows
	// Protected by mu
	// Key: sequence number (calculated from time)
	// Value: window containing request counts
	windows map[int64]*window

	// strictUntil is when this bucket must sync with origin
	// Used after rate limit exceeded to ensure consistency
	strictUntil time.Time
}

// bucketKey uniquely identifies a rate limit bucket by combining the
// identifier, limit, and duration. This ensures separate tracking when
// the same identifier has different rate limit configurations.
//
// Thread Safety:
//   - Immutable after creation
//   - Safe for concurrent use
//
// Example:
//
//	key := bucketKey{
//	    identifier: "user-123",
//	    limit:      100,
//	    duration:   time.Minute,
//	}
//	bucketID := key.toString()
type bucketKey struct {
	// identifier is the rate limit subject (user ID, API key, etc)
	identifier string

	// limit is the maximum requests allowed in the duration
	limit int64

	// duration is the time window for the rate limit
	duration time.Duration
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
func (s *service) getOrCreateBucket(key bucketKey) (*bucket, bool) {

	s.bucketsMu.Lock()
	defer s.bucketsMu.Unlock()
	b, exists := s.buckets[key.toString()]
	if !exists {
		metrics.RatelimitBucketsCreated.Inc()
		b = &bucket{
			mu:          sync.RWMutex{},
			limit:       key.limit,
			duration:    key.duration,
			windows:     make(map[int64]*window),
			strictUntil: time.Time{},
		}
		s.buckets[key.toString()] = b
	}
	return b, exists
}

// getCurrentWindow returns the window for the current time, creating it if needed.
//
// The window's start time is aligned to duration boundaries (e.g., on the minute
// for minute-based limits) to ensure consistent behavior across nodes.
//
// Parameters:
//   - now: Current time to determine the window
//
// Returns:
//   - window: The current window
//   - bool: True if window existed, false if created
//
// Thread Safety:
//   - Caller MUST hold bucket.mu lock
func (b *bucket) getCurrentWindow(now time.Time) (*window, bool) {

	sequence := calculateSequence(now, b.duration)

	w, exists := b.windows[sequence]
	if !exists {
		w = newWindow(sequence, now.Truncate(b.duration), b.duration)
		b.windows[sequence] = w
	}
	return w, exists
}

// getPreviousWindow returns the window immediately before the current one,
// creating it if needed. Used for sliding window calculations.
//
// Parameters:
//   - now: Current time to determine the previous window
//
// Returns:
//   - window: The previous window
//   - bool: True if window existed, false if created
//
// Thread Safety:
//   - Caller MUST hold bucket.mu lock
//
// Performance: O(1) time and space complexity
func (b *bucket) getPreviousWindow(now time.Time) (*window, bool) {

	sequence := calculateSequence(now, b.duration) - 1

	w, exists := b.windows[sequence]
	if !exists {
		w = newWindow(sequence, now.Add(-b.duration).Truncate(b.duration), b.duration)
		b.windows[sequence] = w
	}

	return w, exists
}
