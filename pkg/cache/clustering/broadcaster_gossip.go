package clustering

import (
	"context"
	"sync"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/pkg/cluster"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
)

// CacheInvalidationType is the message type used for cache invalidation
// messages in the cluster message mux.
const CacheInvalidationType = "cache.invalidation"

// GossipBroadcaster implements Broadcaster using the gossip cluster for
// cache invalidation. It serializes protobuf events to raw bytes for the
// cluster layer and deserializes incoming bytes back to events.
type GossipBroadcaster struct {
	mu      sync.RWMutex
	cluster cluster.Cluster
	handler func(context.Context, *cachev1.CacheInvalidationEvent) error
}

var _ Broadcaster = (*GossipBroadcaster)(nil)

// NewGossipBroadcaster creates a new gossip-based broadcaster.
// Call SetCluster after the cluster is created, and Subscribe to register
// the event handler.
func NewGossipBroadcaster() *GossipBroadcaster {
	return &GossipBroadcaster{
		mu:      sync.RWMutex{},
		cluster: nil,
		handler: nil,
	}
}

// OnMessage is the callback wired into cluster.Config.OnMessage.
// It deserializes the raw bytes into a CacheInvalidationEvent and
// dispatches to the registered handler.
func (b *GossipBroadcaster) OnMessage(msg []byte) {
	var event cachev1.CacheInvalidationEvent
	if err := proto.Unmarshal(msg, &event); err != nil {
		logger.Error("Failed to unmarshal gossip cache event", "error", err)
		return
	}

	b.mu.RLock()
	h := b.handler
	b.mu.RUnlock()

	if h != nil {
		if err := h(context.Background(), &event); err != nil {
			logger.Error("Failed to handle gossip cache event", "error", err)
		}
	}
}

// SetCluster wires the broadcaster to a live cluster instance.
// Must be called after cluster.New() returns.
func (b *GossipBroadcaster) SetCluster(c cluster.Cluster) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cluster = c
}

// Broadcast serializes the events and sends them via the gossip cluster.
func (b *GossipBroadcaster) Broadcast(_ context.Context, events ...*cachev1.CacheInvalidationEvent) error {
	b.mu.RLock()
	c := b.cluster
	b.mu.RUnlock()

	if c == nil {
		return nil
	}

	for _, event := range events {
		data, err := proto.Marshal(event)
		if err != nil {
			logger.Error("Failed to marshal cache invalidation event", "error", err)
			continue
		}
		envelope, err := cluster.Wrap(CacheInvalidationType, data)
		if err != nil {
			logger.Error("Failed to wrap cache invalidation envelope", "error", err)
			continue
		}
		if err := c.Broadcast(envelope); err != nil {
			logger.Error("Failed to broadcast cache invalidation", "error", err)
		}
	}

	return nil
}

// Subscribe registers the handler for incoming invalidation events.
func (b *GossipBroadcaster) Subscribe(_ context.Context, handler func(context.Context, *cachev1.CacheInvalidationEvent) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handler = handler
}

// Close shuts down the underlying cluster.
func (b *GossipBroadcaster) Close() error {
	b.mu.RLock()
	c := b.cluster
	b.mu.RUnlock()

	if c != nil {
		return c.Close()
	}
	return nil
}
