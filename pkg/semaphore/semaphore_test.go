package semaphore

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSemaphore_LimitsConcurrency(t *testing.T) {
	const maxConcurrent = 3
	const totalJobs = 20
	sem := New(maxConcurrent)

	var running atomic.Int32
	var maxSeen atomic.Int32
	var wg sync.WaitGroup
	wg.Add(totalJobs)

	for i := 0; i < totalJobs; i++ {
		sem.Do(func() {
			defer wg.Done()
			cur := running.Add(1)
			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			running.Add(-1)
		})
	}

	wg.Wait()

	require.LessOrEqual(t, maxSeen.Load(), int32(maxConcurrent))
	require.Equal(t, int32(0), running.Load())
}

func TestNew_PanicsOnZero(t *testing.T) {
	require.Panics(t, func() { New(0) })
}

func TestNew_PanicsOnNegative(t *testing.T) {
	require.Panics(t, func() { New(-1) })
}
