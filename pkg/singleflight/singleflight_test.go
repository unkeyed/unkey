package singleflight_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/singleflight"
)

func TestDoContext_CanceledWaiterDoesNotCancelSharedCall(t *testing.T) {
	var group singleflight.Group[string]
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	var value string
	var callErr error
	go func() {
		defer close(finished)
		value, callErr = group.DoContext(t.Context(), "shared", func() (string, error) {
			close(started)
			<-release
			return "result", nil
		})
	}()
	t.Cleanup(func() {
		unblock()
		<-finished
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("shared call did not start")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	t.Cleanup(cancel)
	duplicate, err := group.DoContext(ctx, "shared", func() (string, error) {
		t.Error("duplicate call must not execute")
		return "duplicate", nil
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Empty(t, duplicate)

	unblock()
	<-finished
	require.NoError(t, callErr)
	require.Equal(t, "result", value)

	_, err = group.DoContext(ctx, "not-started", func() (string, error) {
		t.Error("already canceled call must not execute")
		return "unexpected", nil
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
