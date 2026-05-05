package bus

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
)

// freePort grabs an ephemeral port and immediately closes the listener. It
// has a small race window before the caller binds, but for in-process tests
// without parallelism around the same port it's reliable enough.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

// newTestBus creates a Serf-backed bus on loopback, joining the supplied
// seeds. Cleanup is registered on the test.
func newTestBus(t *testing.T, nodeID, region string, seeds []string) (Bus, int) {
	t.Helper()

	port := freePort(t)
	b, err := New(Config{
		Region:        region,
		NodeID:        nodeID,
		BindAddr:      "127.0.0.1",
		BindPort:      port,
		AdvertiseAddr: "127.0.0.1",
		Seeds:         seeds,
		// Larger envelope budget so the test cache-key payload fits comfortably
		// alongside the protobuf overhead — keeps the test focused on bus
		// semantics, not envelope sizing.
		MaxUserEventSize: 4096,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = b.Close()
	})
	return b, port
}

// waitForMembers polls Members until size or the deadline. The expected
// count includes the local node.
func waitForMembers(t *testing.T, b Bus, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		got := len(b.Members())
		if got >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("waited %s for %d members, got %d", timeout, want, got)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestPublishSubscribe_DeliversAcrossNodes is the basic end-to-end check:
// a publish on node A is observed by a subscriber on node B.
func TestPublishSubscribe_DeliversAcrossNodes(t *testing.T) {
	a, portA := newTestBus(t, "node-a", "us-east-1", nil)
	seed := fmt.Sprintf("127.0.0.1:%d", portA)
	b, _ := newTestBus(t, "node-b", "eu-central-1", []string{seed})

	waitForMembers(t, a, 2, 5*time.Second)
	waitForMembers(t, b, 2, 5*time.Second)

	received := make(chan Event, 1)
	b.Subscribe("cache.invalidate.test", func(e Event) {
		received <- e
	})

	err := a.Publish(context.Background(), "cache.invalidate.test",
		&cachev1.CacheInvalidationEvent{
			Action: &cachev1.CacheInvalidationEvent_CacheKey{CacheKey: "k1"},
		})
	require.NoError(t, err)

	select {
	case e := <-received:
		require.Equal(t, "cache.invalidate.test", e.Topic)
		require.Equal(t, "node-a", e.SenderNode)
		require.Equal(t, "us-east-1", e.SourceRegion)
		require.NotEmpty(t, e.ID)
	case <-time.After(5 * time.Second):
		t.Fatal("subscriber did not receive event within 5s")
	}
}

// TestPublishSubscribe_SelfDeliveryIsSuppressed pins the contract that a
// publisher does not observe its own events. Without this, every handler
// would have to defensively check SenderNode.
func TestPublishSubscribe_SelfDeliveryIsSuppressed(t *testing.T) {
	a, _ := newTestBus(t, "solo", "us-east-1", nil)

	called := atomic.Int32{}
	a.Subscribe("self.test", func(Event) {
		called.Add(1)
	})

	err := a.Publish(context.Background(), "self.test", &cachev1.CacheInvalidationEvent{
		Action: &cachev1.CacheInvalidationEvent_ClearAll{ClearAll: true},
	})
	require.NoError(t, err)

	// Wait long enough for the user event to round-trip through Serf if
	// self-delivery were not suppressed.
	time.Sleep(500 * time.Millisecond)
	require.Equal(t, int32(0), called.Load(), "publisher must not receive its own events")
}

// TestPublish_PauseDropsAndResumeRestores verifies the kill-switch
// semantics that incident response depends on.
func TestPublish_PauseDropsAndResumeRestores(t *testing.T) {
	a, portA := newTestBus(t, "node-a", "us-east-1", nil)
	seed := fmt.Sprintf("127.0.0.1:%d", portA)
	b, _ := newTestBus(t, "node-b", "eu-central-1", []string{seed})

	waitForMembers(t, a, 2, 5*time.Second)

	var (
		mu       sync.Mutex
		received []string
	)
	b.Subscribe("pause.test", func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e.ID)
	})

	a.Pause()
	err := a.Publish(context.Background(), "pause.test", &cachev1.CacheInvalidationEvent{})
	require.ErrorIs(t, err, ErrBusPaused)

	a.Resume()
	require.NoError(t, a.Publish(context.Background(), "pause.test", &cachev1.CacheInvalidationEvent{}))

	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		got := len(received)
		mu.Unlock()
		if got >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("subscriber did not receive event after Resume")
		}
		time.Sleep(25 * time.Millisecond)
	}

	mu.Lock()
	require.Len(t, received, 1, "exactly one event should have been delivered (the post-Resume publish)")
	mu.Unlock()
}

// TestPublish_PayloadTooLargeReturnsError pins the wire-budget contract.
func TestPublish_PayloadTooLargeReturnsError(t *testing.T) {
	port := freePort(t)
	b, err := New(Config{
		Region:           "us-east-1",
		NodeID:           "tiny",
		BindAddr:         "127.0.0.1",
		BindPort:         port,
		AdvertiseAddr:    "127.0.0.1",
		MaxUserEventSize: 64, // tighter than envelope overhead
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })

	err = b.Publish(context.Background(), "oversize", &cachev1.CacheInvalidationEvent{
		Action: &cachev1.CacheInvalidationEvent_CacheKey{CacheKey: "this-key-pushes-past-the-tiny-budget"},
	})
	require.ErrorIs(t, err, ErrPayloadTooLarge)
}
