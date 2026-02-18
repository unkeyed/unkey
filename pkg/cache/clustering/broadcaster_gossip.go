package clustering

import (
	"context"
	"sync"
	"sync/atomic"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/cluster"
	"github.com/unkeyed/unkey/pkg/logger"
)

// invalidationHandler wraps the handler func so we can use atomic.Pointer
// (atomic.Pointer requires a named type, not a bare func signature).
type invalidationHandler struct {
	fn func(context.Context, *cachev1.CacheInvalidationEvent) error
}

// GossipBroadcaster implements Broadcaster using the gossip cluster for
// cache invalidation. It builds ClusterMessage envelopes with the oneof
// variant directly, avoiding double serialization.
type GossipBroadcaster struct {
	cluster cluster.Cluster
	handler atomic.Pointer[invalidationHandler]

	closeOnce sync.Once
	closeErr  error
}

var _ Broadcaster = (*GossipBroadcaster)(nil)

// NewGossipBroadcaster creates a new gossip-based broadcaster wired to the
// given cluster instance.
func NewGossipBroadcaster(c cluster.Cluster) *GossipBroadcaster {
	return &GossipBroadcaster{
		cluster:   c,
		handler:   atomic.Pointer[invalidationHandler]{},
		closeOnce: sync.Once{},
		closeErr:  nil,
	}
}

// HandleCacheInvalidation is the typed handler for cache invalidation messages.
// Register it with cluster.Subscribe(mux, broadcaster.HandleCacheInvalidation).
func (b *GossipBroadcaster) HandleCacheInvalidation(ci *clusterv1.ClusterMessage_CacheInvalidation) {
	if h := b.handler.Load(); h != nil {
		if err := h.fn(context.Background(), ci.CacheInvalidation); err != nil {
			logger.Error("Failed to handle gossip cache event", "error", err)
		}
	}
}

// Broadcast serializes the events and sends them via the gossip cluster.
func (b *GossipBroadcaster) Broadcast(_ context.Context, events ...*cachev1.CacheInvalidationEvent) error {
	for _, event := range events {
		if err := b.cluster.Broadcast(&clusterv1.ClusterMessage_CacheInvalidation{
			CacheInvalidation: event,
		}); err != nil {
			logger.Error("Failed to broadcast cache invalidation", "error", err)
		}
	}

	return nil
}

// Subscribe sets the single handler for incoming invalidation events.
// Calling Subscribe again replaces the previous handler.
func (b *GossipBroadcaster) Subscribe(_ context.Context, handler func(context.Context, *cachev1.CacheInvalidationEvent) error) {
	b.handler.Store(&invalidationHandler{fn: handler})
}

// Close shuts down the underlying cluster. It is safe to call multiple times;
// only the first call closes the cluster, subsequent calls return the original result.
func (b *GossipBroadcaster) Close() error {
	b.closeOnce.Do(func() {
		b.closeErr = b.cluster.Close()
	})
	return b.closeErr
}
