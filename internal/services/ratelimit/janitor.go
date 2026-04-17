package ratelimit

import (
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"github.com/unkeyed/unkey/pkg/repeat"
)

// startJanitor schedules runJanitorOnce to run every minute.
func (s *service) startJanitor() {
	repeat.Every(time.Minute, s.runJanitorOnce)
}

// runJanitorOnce performs a single cleanup pass. It prevents unbounded memory
// growth from two sources:
//
//   - Sliding-window counters whose window ended more than 3x its duration ago.
//   - Strict-mode deadlines that are already in the past.
//
// Uses sync.Map.Range + CompareAndDelete so cleanup never blocks rate limit
// operations.
func (s *service) runJanitorOnce() {
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

	nowMs := now.UnixMilli()
	s.strictUntils.Range(func(key, value any) bool {
		if value.(*atomic.Int64).Load() < nowMs {
			s.strictUntils.CompareAndDelete(key, value)
		}
		return true
	})
}
