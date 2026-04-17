package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
)

// TestJanitor_EvictsExpiredCounters asserts counters whose window ended more
// than 3x the duration ago are removed, while fresh counters survive.
func TestJanitor_EvictsExpiredCounters(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	svc, err := New(Config{Clock: clk, Counter: counter.NewMemory()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	duration := time.Minute
	durationMs := duration.Milliseconds()

	// Counter for a window that ended 10 minutes ago (far past 3× duration).
	oldSeq := calculateSequence(clk.Now().Add(-10*time.Minute), duration)
	oldKey := counterKey{name: "ns", identifier: "expired", durationMs: durationMs, sequence: oldSeq}
	svc.counters.Store(oldKey, &counterEntry{}) //nolint:exhaustruct

	// Counter for the current window — should survive.
	freshSeq := calculateSequence(clk.Now(), duration)
	freshKey := counterKey{name: "ns", identifier: "fresh", durationMs: durationMs, sequence: freshSeq}
	svc.counters.Store(freshKey, &counterEntry{}) //nolint:exhaustruct

	svc.runJanitorOnce()

	_, oldStillExists := svc.counters.Load(oldKey)
	_, freshStillExists := svc.counters.Load(freshKey)
	require.False(t, oldStillExists, "expired counter should be evicted")
	require.True(t, freshStillExists, "current counter should survive")
}

// TestJanitor_EvictsExpiredStrictUntils asserts strict-mode deadlines in the
// past are removed while future deadlines are kept.
func TestJanitor_EvictsExpiredStrictUntils(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	svc, err := New(Config{Clock: clk, Counter: counter.NewMemory()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	durationMs := time.Minute.Milliseconds()

	pastKey := strictKey{name: "ns", identifier: "past", durationMs: durationMs}
	svc.setStrictUntil(pastKey, clk.Now().Add(-time.Minute).UnixMilli())

	futureKey := strictKey{name: "ns", identifier: "future", durationMs: durationMs}
	svc.setStrictUntil(futureKey, clk.Now().Add(time.Minute).UnixMilli())

	svc.runJanitorOnce()

	_, pastStillExists := svc.strictUntils.Load(pastKey)
	_, futureStillExists := svc.strictUntils.Load(futureKey)
	require.False(t, pastStillExists, "past deadline should be evicted")
	require.True(t, futureStillExists, "future deadline should survive")
}
