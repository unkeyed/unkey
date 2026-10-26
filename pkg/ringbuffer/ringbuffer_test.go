package ringbuffer

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPushOverwritesOldestFirst(t *testing.T) {
	r := New[string](2)
	_, full := r.Push("a")
	require.False(t, full)
	_, full = r.Push("b")
	require.False(t, full)

	for _, push := range [][2]string{{"c", "a"}, {"d", "b"}, {"e", "c"}} {
		overwritten, full := r.Push(push[0])
		require.True(t, full)
		require.Equal(t, push[1], overwritten)
	}
}

func TestZeroCapacityReturnsEveryPush(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		r := New[int](capacity)
		overwritten, full := r.Push(7)
		require.True(t, full)
		require.Equal(t, 7, overwritten)
	}
}

// TestRingMatchesSlice runs seeded pushes and clears against a slice that
// drops its first item when it grows past capacity.
func TestRingMatchesSlice(t *testing.T) {
	for _, capacity := range []int{1, 2, 3, 10} {
		rng := rand.New(rand.NewPCG(uint64(capacity), 0))
		r := New[int](capacity)
		var want []int
		for i := range 10_000 {
			if rng.IntN(100) == 0 {
				r.Clear()
				want = nil
				continue
			}
			overwritten, full := r.Push(i)
			want = append(want, i)
			require.Equal(t, len(want) > capacity, full, "push %d", i)
			if len(want) > capacity {
				require.Equal(t, want[0], overwritten, "push %d", i)
				want = want[1:]
			}
		}
	}
}
