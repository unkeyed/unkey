//go:build linux

package spool_test

import (
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
)

// TestBody_ReadersAreIndependent guarantees that each forwarding attempt can
// replay the whole body from the start, even if an earlier reader stopped
// part way.
func TestBody_ReadersAreIndependent(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t, spool.Config{Dir: "", BodyBytesMax: 1 << 20, TotalBytesMax: 1 << 20, ConcurrentMax: 1})
	data := patternBytes(70_001)
	body := admitBytes(t, pool, data)
	t.Cleanup(func() { require.NoError(t, body.Close()) })

	first := body.Reader()
	_, err := io.ReadFull(first, make([]byte, 1000))
	require.NoError(t, err)

	requireBodyBytes(t, body, data)
	rest, err := io.ReadAll(first)
	require.NoError(t, err)
	require.Equal(t, data[1000:], rest)
}

// TestBody_ConcurrentCloseReleasesOnce guarantees that closing a body from
// several goroutines, while readers are active, closes the file and returns
// its slot and bytes exactly once.
func TestBody_ConcurrentCloseReleasesOnce(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 64 << 10, TotalBytesMax: 128 << 10, ConcurrentMax: 2}
	pool := newTestPool(t, cfg)
	body := admitBytes(t, pool, patternBytes(64<<10))

	closeErrs := make(chan error, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			// A reader may see the file closed mid-read; only Close must succeed.
			_, _ = io.Copy(io.Discard, body.Reader())
			closeErrs <- body.Close()
		})
	}
	wg.Wait()
	close(closeErrs)

	for err := range closeErrs {
		require.NoError(t, err)
	}
	requireExactCapacity(t, pool, cfg)
}

// TestBody_ReaderFailsAfterClose guarantees that a reader that was not
// exhausted cannot read file data after its owner closed the body.
func TestBody_ReaderFailsAfterClose(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t, spool.Config{Dir: "", BodyBytesMax: 8, TotalBytesMax: 8, ConcurrentMax: 1})
	body := admitBytes(t, pool, []byte("abc"))
	reader := body.Reader()

	require.NoError(t, body.Close())

	_, err := reader.Read(make([]byte, 3))
	require.ErrorIs(t, err, os.ErrClosed)
}
