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

// HandleClusterMessage is the callback registered with MessageMux.Subscribe.
// It extracts cache invalidation events from the cluster envelope and
// dispatches them to the registered handler.
func (b *GossipBroadcaster) HandleClusterMessage(msg *clusterv1.ClusterMessage) {
	ci, ok := msg.Message.(*clusterv1.ClusterMessage_CacheInvalidation)
	if !ok {
		return
	}

	if b.handler != nil {
		if err := b.handler(context.Background(), ci.CacheInvalidation); err != nil {
			logger.Error("Failed to handle gossip cache event", "error", err)
		}
	}
}

// Broadcast serializes the events and sends them via the gossip cluster.
func (b *GossipBroadcaster) Broadcast(_ context.Context, events ...*cachev1.CacheInvalidationEvent) error {
	for _, event := range events {
		if err := b.cluster.Broadcast(&clusterv1.ClusterMessage{
			Message: &clusterv1.ClusterMessage_CacheInvalidation{
				CacheInvalidation: event,
			},
		}); err != nil {
			logger.Error("Failed to broadcast cache invalidation", "error", err)
		}
	}

	return nil
}

// Subscribe registers the handler for incoming invalidation events.
func (b *GossipBroadcaster) Subscribe(_ context.Context, handler func(context.Context, *cachev1.CacheInvalidationEvent) error) {
	b.handler = handler
}

// Close shuts down the underlying cluster.
func (b *GossipBroadcaster) Close() error {
	return b.cluster.Close()
}
