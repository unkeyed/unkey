package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNew verifies that a fresh Runner starts in a clean state with no tasks
// or cleanups registered and neither running nor shutting down.
func TestNew(t *testing.T) {
	r := New()
	require.NotNil(t, r)
	require.Empty(t, r.tasks)
	require.Empty(t, r.cleanups)
	require.False(t, r.running)
	require.False(t, r.shuttingDown)
}

// TestGo_RegistersTask verifies that Go adds a task to the runner's task list.
func TestGo_RegistersTask(t *testing.T) {
	r := New()

	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	r.mu.Lock()
	require.Len(t, r.tasks, 1)
	r.mu.Unlock()
}

// TestDefer_RegistersCleanup verifies that Defer adds a cleanup function and
// that it gets called during shutdown.
func TestDefer_RegistersCleanup(t *testing.T) {
	r := New()

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

	err := r.Run(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.True(t, called)
}

// TestDeferCtx_RegistersCleanup verifies that DeferCtx adds a context-aware
// cleanup function and that it gets called during shutdown.
func TestDeferCtx_RegistersCleanup(t *testing.T) {
	r := New()

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

	err := r.Run(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.True(t, called)
}

// TestRegister_AliasForDefer verifies that Register behaves identically to
// Defer. This alias exists for interface compatibility with existing shutdown
// patterns.
func TestRegister_AliasForDefer(t *testing.T) {
	r := New()

	called := false
	r.Register(func() error {
		called = true
		return nil
	})

	r.mu.Lock()
	require.Len(t, r.cleanups, 1)
	r.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Run(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.True(t, called)
}

// TestRegisterCtx_AliasForDeferCtx verifies that RegisterCtx behaves
// identically to DeferCtx. This alias exists for compatibility with the
// otel.Registrar interface.
func TestRegisterCtx_AliasForDeferCtx(t *testing.T) {
	r := New()

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

	err := r.Run(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.True(t, called)
}
