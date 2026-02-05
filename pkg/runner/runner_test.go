package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// TestNew verifies that a fresh Runner starts in a clean state with no
// cleanups registered and neither running nor shutting down.
func TestNew(t *testing.T) {
	r := New(logging.NewNoop())
	require.NotNil(t, r)
	require.Empty(t, r.cleanups)
	require.Equal(t, runnerStateIdle, r.state)
	require.NotNil(t, r.ctx)
	require.NotNil(t, r.cancel)
}

// TestGo_StartsImmediately verifies that Go starts the task immediately.
func TestGo_StartsImmediately(t *testing.T) {
	r := New()

	started := make(chan struct{})
	r.Go(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return nil
	})

	select {
	case <-started:
		// Good - task started immediately
	case <-time.After(100 * time.Millisecond):
		t.Fatal("task did not start immediately")
	}

	// Cleanup
	r.cancel()
	r.wg.Wait()
}

// TestDefer_RegistersCleanup verifies that Defer adds a cleanup function and
// that it gets called during shutdown.
func TestDefer_RegistersCleanup(t *testing.T) {
	r := New(logging.NewNoop())

	called := false
	r.Defer(func() error {
		called = true
		return nil
	})

	r.mu.Lock()
	require.Len(t, r.cleanups, 1)
	r.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.True(t, called)
}

// TestDeferCtx_RegistersCleanup verifies that DeferCtx adds a context-aware
// cleanup function and that it gets called during shutdown.
func TestDeferCtx_RegistersCleanup(t *testing.T) {
	r := New(logging.NewNoop())

	called := false
	r.DeferCtx(func(ctx context.Context) error {
		called = true
		return nil
	})

	r.mu.Lock()
	require.Len(t, r.cleanups, 1)
	r.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.True(t, called)
}

// TestRegisterCtx_AliasForDeferCtx verifies that RegisterCtx behaves
// identically to DeferCtx. This alias exists for compatibility with the
// otel.Registrar interface.
func TestRegisterCtx_AliasForDeferCtx(t *testing.T) {
	r := New(logging.NewNoop())

	called := false
	r.RegisterCtx(func(ctx context.Context) error {
		called = true
		return nil
	})

	r.mu.Lock()
	require.Len(t, r.cleanups, 1)
	r.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.True(t, called)
}
