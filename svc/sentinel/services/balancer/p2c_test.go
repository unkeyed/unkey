package balancer

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestP2CBalancer_SingleInstance(t *testing.T) {
	b := NewP2CBalancer()
	ids := []string{"a"}

	for range 100 {
		require.Equal(t, 0, b.Pick(ids))
	}
}

func TestP2CBalancer_PrefersLeastLoaded(t *testing.T) {
	b := NewP2CBalancer()

	// Simulate 10 in-flight requests on instance "a", 0 on "b".
	for range 10 {
		b.Acquire("a")
	}

	ids := []string{"a", "b"}
	pickedB := 0
	const trials = 1000
	for range trials {
		if b.Pick(ids) == 1 {
			pickedB++
		}
	}

	// With 10 vs 0 inflight, P2C should pick "b" nearly always.
	// Allow some slack for randomness but it should be >90%.
	require.Greater(t, pickedB, int(float64(trials)*0.9),
		"expected P2C to strongly prefer the less-loaded instance")
}

func TestP2CBalancer_DistributesEvenly(t *testing.T) {
	b := NewP2CBalancer()
	ids := []string{"a", "b", "c"}

	counts := map[int]int{}
	const trials = 10000
	for range trials {
		idx := b.Pick(ids)
		counts[idx]++
	}

	// With equal load, distribution should be roughly uniform.
	for i, c := range counts {
		ratio := float64(c) / float64(trials)
		require.InDelta(t, 1.0/3.0, ratio, 0.05,
			"instance %d picked %.1f%% of the time, expected ~33%%", i, ratio*100)
	}
}

func TestP2CBalancer_AcquireRelease(t *testing.T) {
	b := NewP2CBalancer()

	b.Acquire("a")
	b.Acquire("a")
	b.Acquire("a")
	require.Equal(t, int64(3), b.Inflight("a"))

	b.Release("a")
	require.Equal(t, int64(2), b.Inflight("a"))

	b.Release("a")
	b.Release("a")
	require.Equal(t, int64(0), b.Inflight("a"))
}

func TestP2CBalancer_ConcurrentAccess(t *testing.T) {
	b := NewP2CBalancer()
	ids := []string{"a", "b", "c", "d"}

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				idx := b.Pick(ids)
				id := ids[idx]
				b.Acquire(id)
				b.Release(id)
			}
		}()
	}
	wg.Wait()

	// After all goroutines complete, all inflight counts should be 0.
	for _, id := range ids {
		require.Equal(t, int64(0), b.Inflight(id))
	}
}
