package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// replayUpdate carries the result of an async Redis Increment back to the
// bucket without requiring the caller to hold bucket.mu.
type replayUpdate struct {
	sequence   int64
	newCounter int64
}

// bucket maintains rate limit state for a specific identifier+duration combination.
// It stores a sliding window of request counts and manages the lifecycle of these windows.
//
// Each bucket is uniquely identified by a pair of:
//   - identifier: The rate limit subject (user ID, API key, etc)
//   - duration: Time window for the rate limit
//
// Example Usage:
//
//	b := &bucket{
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

	// name is the name of the bucket
	name string

	// identifier is the rate limit subject (user ID, API key, etc)
	identifier string

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

	// updates receives counter updates from replay workers without
	// requiring them to hold mu. Drained by Ratelimit() callers who
	// already hold the lock.
	updates chan replayUpdate
}

func (b *bucket) key() bucketKey {
	return bucketKey{
		name:       b.name,
		identifier: b.identifier,
		duration:   b.duration,
	}
}

// bucketKey uniquely identifies a rate limit bucket by combining the
// identifier and duration.
//
// Thread Safety:
//   - Immutable after creation
//   - Safe for concurrent use
//
// Example:
//
//	key := bucketKey{
//	    identifier: "user-123",
//	    duration:   time.Minute,
//	}
//	bucketID := key.toString()
type bucketKey struct {
	// name is an arbitrary name for the bucket
	name string

	// identifier is the rate limit subject (user ID, API key, etc)
	identifier string

	// duration is the time window for the rate limit
	duration time.Duration
}

func (b bucketKey) toString() string {
	return fmt.Sprintf("%s-%s-%d", b.name, b.identifier, b.duration.Milliseconds())
}

// getOrCreateBucket retrieves a rate limiting bucket for the given key.
// If no bucket exists, it creates a new one.
//
// The bucket is uniquely identified by the combination of:
// - identifier: the client identifier being rate limited
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
			name:        key.name,
			identifier:  key.identifier,
			duration:    key.duration,
			windows:     make(map[int64]*window),
			strictUntil: time.Time{},
			updates:     make(chan replayUpdate, 8),
		}
		s.buckets[key.toString()] = b
	}

	return b, exists
}

// drainUpdates applies any pending replay updates to the bucket's windows.
// The caller MUST hold bucket.mu.Lock() before calling this.
func (b *bucket) drainUpdates() {
	for {
		select {
		case u := <-b.updates:
			w, exists := b.windows[u.sequence]
			if !exists {
				// Window was already expired/cleaned up, skip.
				continue
			}
			if u.newCounter > w.counter {
				w.counter = u.newCounter
			}
		default:
			return
		}
	}
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
