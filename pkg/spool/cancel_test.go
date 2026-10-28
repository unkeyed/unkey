//go:build linux

package spool_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
)

// TestAdmit_CancelUnblocksBlockedRead guarantees that canceling ctx ends an
// admission whose input read is blocked, by closing the input, and that the
// admission then releases everything.
func TestAdmit_CancelUnblocksBlockedRead(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 64, TotalBytesMax: 64, ConcurrentMax: 1}
	pool := newTestPool(t, cfg)
	reader, writer := io.Pipe()
	in := newInput(reader)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	result := make(chan error, 1)
	go func() {
		body, err := pool.Admit(ctx, in, -1, readAll)
		if body != nil {
			err = errors.Join(errors.New("admitted a canceled body"), body.Close())
		}
		result <- err
	}()
	_, err := writer.Write([]byte("abc"))
	require.NoError(t, err)
	cancel()

	select {
	case err = <-result:
	case <-time.After(10 * time.Second):
		t.Fatal("Admit did not return after cancellation")
	}
	requireKind(t, err, spool.FailureCanceled)
	_, err = writer.Write([]byte("d"))
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.Equal(t, int64(1), in.closes.Load())
	requireExactCapacity(t, pool, cfg)
}

// TestAdmit_CancelAfterValidationPreventsCommit guarantees that a context
// canceled after the validator accepted the body, but before the commit
// point, yields no body.
func TestAdmit_CancelAfterValidationPreventsCommit(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 8, TotalBytesMax: 8, ConcurrentMax: 1}
	pool := newTestPool(t, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	in := newInput(strings.NewReader("abc"))

	body, err := pool.Admit(ctx, in, 3, func(ctx context.Context, r io.Reader) error {
		readErr := readAll(ctx, r)
		cancel()
		return readErr
	})

	require.Nil(t, body)
	requireKind(t, err, spool.FailureCanceled)
	require.Equal(t, int64(1), in.closes.Load())
	requireExactCapacity(t, pool, cfg)
}

// TestAdmit_CancellationRacingCommit guarantees that a cancellation racing the
// commit point either prevents the body or leaves a complete, readable body:
// cancellation after the commit point must not close or truncate it.
func TestAdmit_CancellationRacingCommit(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 8, TotalBytesMax: 8, ConcurrentMax: 1}
	pool := newTestPool(t, cfg)

	for range 200 {
		ctx, cancel := context.WithCancel(context.Background())
		in := newInput(strings.NewReader("abc"))
		go cancel()

		body, err := pool.Admit(ctx, in, 3, readAll)

		<-ctx.Done()
		if err != nil {
			requireKind(t, err, spool.FailureCanceled)
		} else {
			requireBodyBytes(t, body, []byte("abc"))
			require.NoError(t, body.Close())
		}
		require.Equal(t, int64(1), in.closes.Load())
	}
	requireExactCapacity(t, pool, cfg)
}

// TestAdmit_EOFRacingCancellation guarantees that when the input reaches EOF
// while ctx is canceled, the result is either the complete body or a
// cancellation, the input is closed once, and nothing leaks.
func TestAdmit_EOFRacingCancellation(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 128, TotalBytesMax: 128, ConcurrentMax: 1}
	pool := newTestPool(t, cfg)
	data := patternBytes(100)

	for range 200 {
		ctx, cancel := context.WithCancel(context.Background())
		reader, writer := io.Pipe()
		in := newInput(reader)
		written := make(chan error, 1)
		go func() {
			_, err := writer.Write(data)
			written <- errors.Join(err, writer.Close())
		}()
		go cancel()

		body, err := pool.Admit(ctx, in, int64(len(data)), readAll)

		writeErr := <-written
		if writeErr != nil {
			require.ErrorIs(t, writeErr, io.ErrClosedPipe)
		}
		if err != nil {
			requireKind(t, err, spool.FailureCanceled)
		} else {
			require.NoError(t, writeErr)
			requireBodyBytes(t, body, data)
			require.NoError(t, body.Close())
		}
		require.Equal(t, int64(1), in.closes.Load())
		cancel()
	}
	requireExactCapacity(t, pool, cfg)
}
