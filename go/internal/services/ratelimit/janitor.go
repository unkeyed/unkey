package ratelimit

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/go/pkg/repeat"
)

// expireWindowsAndBuckets runs a periodic cleanup of expired rate limit windows
// and empty buckets. It prevents unbounded memory growth by removing state that
// is no longer needed for rate limit decisions.
//
// The janitor runs every minute and:
// 1. Removes windows that are older than 3x their duration
// 2. Removes buckets that have no remaining windows
// 3. Updates metrics for monitoring
//
// Thread Safety:
//   - Safe for concurrent access with rate limit operations
//   - Uses appropriate locks to prevent race conditions
//
// Performance:
//   - O(n) where n is total number of windows
//   - Runs in background goroutine
//   - Minimal impact on rate limit operations
//
// Memory Management:
//   - Prevents memory leaks from abandoned rate limits
//   - Gracefully handles varying load patterns
//   - Maintains optimal memory usage over time
//
// Example lifecycle:
//   - Window created at t=0 with 1-minute duration
//   - Window becomes inactive after 1 minute
//   - Window removed by janitor after 3 minutes
//   - Bucket removed when last window expires
func (s *service) expireWindowsAndBuckets() {

	repeat.Every(time.Minute, func() {
		s.bucketsMu.Lock()
		defer s.bucketsMu.Unlock()

		windows := float64(0)

		for bucketID, bucket := range s.buckets {
			bucket.mu.Lock()
			for sequence, window := range bucket.windows {
				if s.clock.Now().After(window.start.Add(3 * window.duration)) {
					delete(bucket.windows, sequence)
					metrics.RatelimitWindowsEvicted.Inc()
				} else {
					windows++
				}
			}
			if len(bucket.windows) == 0 {
				delete(s.buckets, bucketID)
				metrics.RatelimitBucketsEvicted.Inc()
			}

			bucket.mu.Unlock()
		}

		metrics.RatelimitBuckets.Set(float64(len(s.buckets)))
		metrics.RatelimitWindows.Set(windows)
	})

}
