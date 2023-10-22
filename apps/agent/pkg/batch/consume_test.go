package batch_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
)

func TestProcess(t *testing.T) {
	lock := sync.Mutex{}
	handled := make([]int, 0)
	flushed := 0

	// if we don't make a copy under a lock, we get some nasty race conditions in the test
	copy := func() []int {
		lock.Lock()
		defer lock.Unlock()
		return append([]int{}, handled...)
	}

	flush := func(ctx context.Context, batch []int) {
		lock.Lock()
		defer lock.Unlock()
		flushed++
		handled = append(handled, batch...)
	}

	// Create a channel and start processing
	size := 3
	interval := time.Millisecond * 100
	c := batch.Process(flush, size, interval)

	// Send some elements to the channel
	c <- 1
	c <- 2

	// Wait for a bit to let the batch fill up
	time.Sleep(interval / 2)

	// Send another element to the channel
	c <- 3

	// Wait for the batch to be flushed
	time.Sleep(interval)

	// Check that the batches were flushed correctly

	require.Contains(t, copy(), 1)
	require.Contains(t, copy(), 2)
	require.Contains(t, copy(), 3)
	require.Equal(t, 1, flushed)

	// Send some more elements to the channel
	c <- 4
	c <- 5

	time.Sleep(interval)
	// Check that the handled were flushed correctly

	require.Contains(t, copy(), 4)
	require.Contains(t, copy(), 5)
	require.Equal(t, 2, flushed)

	c <- 6
	// Close the channel
	close(c)

	time.Sleep(interval)

	require.Contains(t, copy(), 6)
	require.Equal(t, 3, flushed)

}
