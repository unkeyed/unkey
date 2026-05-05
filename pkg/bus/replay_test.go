package bus

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"google.golang.org/protobuf/proto"
)

// TestReplayOnJoin_NewPeerReceivesPriorEvents covers the partition-recovery
// scenario the replay protocol exists for: node A publishes events while B
// is not yet in the cluster, then B joins and must observe the prior events
// via the replay-on-join query.
//
// The receiver-side dedup guarantees that subsequent gossip retransmits
// don't double-deliver, but that's tested in the live publish-subscribe
// path. Here we assert the recovery path itself.
func TestReplayOnJoin_NewPeerReceivesPriorEvents(t *testing.T) {
	a, portA := newTestBus(t, "node-a", "us-east-1", nil)

	// Publish a few events into node A's replay log before any peer exists.
	// dedup is per-process; each event has a unique id, so the receiver
	// will see each exactly once.
	for i := 0; i < 3; i++ {
		err := a.Publish(context.Background(), "cache.invalidate.test",
			&cachev1.CacheInvalidationEvent{
				Action: &cachev1.CacheInvalidationEvent_CacheKey{
					CacheKey: fmt.Sprintf("k%d", i),
				},
			})
		require.NoError(t, err)
	}

	var (
		mu       sync.Mutex
		received []string
	)
	seed := fmt.Sprintf("127.0.0.1:%d", portA)
	b, _ := newTestBus(t, "node-b", "eu-central-1", []string{seed})

	// Subscribe before joining is processed: NotifyJoin runs through the
	// event loop after Subscribe returns, so the handler is in place by
	// the time replay envelopes get re-dispatched.
	b.Subscribe("cache.invalidate.test", func(e Event) {
		var inv cachev1.CacheInvalidationEvent
		if err := proto.Unmarshal(e.Payload, &inv); err != nil {
			t.Errorf("unmarshal payload: %v", err)
			return
		}
		mu.Lock()
		received = append(received, inv.GetCacheKey())
		mu.Unlock()
	})

	deadline := time.Now().Add(10 * time.Second)
	for {
		mu.Lock()
		got := len(received)
		mu.Unlock()
		if got >= 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 3 replayed events, got %d", got)
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	require.ElementsMatch(t, []string{"k0", "k1", "k2"}, received)
}
