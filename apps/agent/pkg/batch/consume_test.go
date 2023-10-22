package batch_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"golang.org/x/exp/slices"
)

func TestProcess(t *testing.T) {
	// Define a flush function that just appends the batch to a slice
	handled := make([]int, 0)
	flushed := 0

	flush := func(ctx context.Context, batch []int) {
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
	require.Contains(t, handled, 1)
	require.Contains(t, handled, 2)
	require.Contains(t, handled, 3)
	require.Equal(t, 1, flushed)

	// Send some more elements to the channel
	c <- 4
	c <- 5

	// Check that the handled were flushed correctly
	require.Eventually(t, func() bool {
		return slices.Contains(handled, 4)
	}, interval*2, interval/10)
	require.Eventually(t, func() bool {
		return slices.Contains(handled, 5)
	}, interval*2, interval/10)

	c <- 6
	// Close the channel
	close(c)

	require.Eventually(t, func() bool {
		return slices.Contains(handled, 6) && flushed == 3
	}, interval*2, interval/10, "handled should contain 6 and flushed should be 3, but got %v and %v", handled, flushed)

}
