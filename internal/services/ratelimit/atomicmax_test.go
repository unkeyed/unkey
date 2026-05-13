package ratelimit

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtomicMax_RaisesOrNoOps(t *testing.T) {
	tests := []struct {
		name string
		init int64
		val  int64
		want int64
	}{
		{name: "zero to positive raises", init: 0, val: 5, want: 5},
		{name: "positive to higher raises", init: 5, val: 10, want: 10},
		{name: "lower val is a no-op", init: 5, val: 3, want: 5},
		{name: "equal val is a no-op", init: 5, val: 5, want: 5},
		{name: "negative to higher raises", init: -10, val: -5, want: -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ptr := &atomic.Int64{}
			ptr.Store(tt.init)
			atomicMax(ptr, tt.val)
			require.Equal(t, tt.want, ptr.Load())
		})
	}
}

// TestAtomicMax_ConcurrentConverges asserts the final value equals the maximum
// of every proposed value even when many goroutines race. This is the invariant
// the replay merge and strictUntil write paths depend on.
func TestAtomicMax_ConcurrentConverges(t *testing.T) {
	t.Parallel()

	vals := []int64{1, 50, 99, 42, 7, 1000, 33, 500, 2, 777}
	ptr := &atomic.Int64{}
	var wg sync.WaitGroup

	for _, v := range vals {
		wg.Go(func() {
			atomicMax(ptr, v)
		})
	}
	wg.Wait()

	require.Equal(t, slices.Max(vals), ptr.Load())
}
