package clustering

import (
	"context"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/cluster"
	"github.com/unkeyed/unkey/pkg/logger"
)

// GossipBroadcaster implements Broadcaster using the gossip cluster for
// cache invalidation. It builds ClusterMessage envelopes with the oneof
// variant directly, avoiding double serialization.
type GossipBroadcaster struct {
	cluster cluster.Cluster
	handler func(context.Context, *cachev1.CacheInvalidationEvent) error
}

var _ Broadcaster = (*GossipBroadcaster)(nil)

// NewGossipBroadcaster creates a new gossip-based broadcaster wired to the
// given cluster instance.
func NewGossipBroadcaster(c cluster.Cluster) *GossipBroadcaster {
	return &GossipBroadcaster{
		cluster: c,
		handler: nil,
	}
}

// HandleCacheInvalidation is the typed handler for cache invalidation messages.
// Register it with cluster.Subscribe(mux, broadcaster.HandleCacheInvalidation).
func (b *GossipBroadcaster) HandleCacheInvalidation(ci *clusterv1.ClusterMessage_CacheInvalidation) {
	if b.handler != nil {
		if err := b.handler(context.Background(), ci.CacheInvalidation); err != nil {
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
	b.handler = handler
}

// Close shuts down the underlying cluster.
func (b *GossipBroadcaster) Close() error {
	return b.cluster.Close()
}
