package db

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParallelGroup_AllSuccess(t *testing.T) {
	g := NewParallelGroup(context.Background())

	var s string
	var n int
	var bs []byte

	Go(g, &s, func(ctx context.Context) (string, error) { return "hello", nil })
	Go(g, &n, func(ctx context.Context) (int, error) { return 42, nil })
	Go(g, &bs, func(ctx context.Context) ([]byte, error) { return []byte{1, 2, 3}, nil })

	require.NoError(t, g.Wait())
	require.Equal(t, "hello", s)
	require.Equal(t, 42, n)
	require.Equal(t, []byte{1, 2, 3}, bs)
}

func TestParallelGroup_FirstErrorCancelsSiblings(t *testing.T) {
	g := NewParallelGroup(context.Background())

	wantErr := errors.New("boom")

	var fast string
	var slow string
	var siblingObservedCancel atomic.Bool

	Go(g, &fast, func(ctx context.Context) (string, error) {
		return "", wantErr
	})

	Go(g, &slow, func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			siblingObservedCancel.Store(true)
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
			return "should-not-happen", nil
		}
	})

	err := g.Wait()
	require.ErrorIs(t, err, wantErr)
	require.True(t, siblingObservedCancel.Load(), "sibling should observe context cancellation")
	require.Empty(t, slow, "slow result must not be written when its goroutine errored")
}

func TestParallelGroup_ParentCtxCancelStopsAll(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	g := NewParallelGroup(parentCtx)

	var observed atomic.Int32

	for i := 0; i < 3; i++ {
		var out int
		Go(g, &out, func(ctx context.Context) (int, error) {
			select {
			case <-ctx.Done():
				observed.Add(1)
				return 0, ctx.Err()
			case <-time.After(2 * time.Second):
				return 0, nil
			}
		})
	}

	cancel()

	err := g.Wait()
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(3), observed.Load())
}

func TestParallelGroup_OutPointerOnlyWrittenOnSuccess(t *testing.T) {
	g := NewParallelGroup(context.Background())

	preset := "untouched"
	Go(g, &preset, func(ctx context.Context) (string, error) {
		return "new value", errors.New("nope")
	})

	require.Error(t, g.Wait())
	require.Equal(t, "untouched", preset, "out pointer must not be written when fn returns an error")
}
