package ratelimit

import (
	"time"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"github.com/unkeyed/unkey/pkg/repeat"
)

// startJanitor runs a periodic cleanup of expired counters.
// It prevents unbounded memory growth by removing counters for windows that
// are older than 3x their duration.
//
// Uses sync.Map.Range + CompareAndDelete so cleanup never blocks rate limit
// operations.
func (s *service) startJanitor() {
	repeat.Every(time.Minute, func() {
		now := s.clock.Now()
		activeCounters := float64(0)

		s.counters.Range(func(key, value any) bool {
			k := key.(counterKey)

			windowStartMs := k.sequence * k.durationMs
			duration := time.Duration(k.durationMs) * time.Millisecond

			if now.After(time.UnixMilli(windowStartMs).Add(3 * duration)) {
				s.counters.CompareAndDelete(key, value)
				metrics.RatelimitWindowsEvicted.Inc()
			} else {
				activeCounters++
			}
			return true
		})

		metrics.RatelimitWindows.Set(activeCounters)
	})
}
