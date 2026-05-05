package bus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
)

// The noop bus exists so callers can defer Subscribe's unsubscribe and call
// Query in a range loop without checking which Bus implementation they hold.
// These tests pin those contracts.

func TestNoop_PublishReturnsNil(t *testing.T) {
	b := NewNoop()

	err := b.Publish(context.Background(), "any", &cachev1.CacheInvalidationEvent{})
	require.NoError(t, err)
}

func TestNoop_SubscribeReturnsCallableUnsubscribe(t *testing.T) {
	b := NewNoop()

	unsub := b.Subscribe("any", func(Event) {
		t.Fatal("noop bus must never invoke handlers")
	})
	require.NotNil(t, unsub)
	unsub()
	unsub() // double-unsubscribe must be safe
}

func TestNoop_QueryChannelClosedImmediately(t *testing.T) {
	b := NewNoop()

	ch, err := b.Query(context.Background(), "any", &cachev1.CacheInvalidationEvent{})
	require.NoError(t, err)

	// A range loop over the noop query channel must terminate without
	// blocking. If the channel were not closed, this test would hang.
	count := 0
	for range ch {
		count++
	}
	require.Equal(t, 0, count)
}

func TestNoop_MembersIsNil(t *testing.T) {
	b := NewNoop()
	require.Nil(t, b.Members())
}

func TestNoop_PauseResumeCloseAreSafe(t *testing.T) {
	b := NewNoop()

	b.Pause()
	b.Resume()
	require.NoError(t, b.Close())
	require.NoError(t, b.Close()) // idempotent
}
