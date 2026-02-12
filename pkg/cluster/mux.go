package cluster

import (
	"sync"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
)

// MessageMux routes incoming cluster messages to registered handlers based on
// the envelope's oneof variant. It sits between the cluster transport and
// application-level handlers, allowing multiple subsystems to share the same
// gossip cluster.
type MessageMux struct {
	mu                       sync.RWMutex
	cacheInvalidationHandler func(*cachev1.CacheInvalidationEvent)
}

// NewMessageMux creates a new message multiplexer.
func NewMessageMux() *MessageMux {
	return &MessageMux{}
}

// HandleCacheInvalidation registers a handler for cache invalidation messages.
func (m *MessageMux) HandleCacheInvalidation(handler func(*cachev1.CacheInvalidationEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheInvalidationHandler = handler
}

// OnMessage deserializes the ClusterMessage envelope and dispatches to the
// handler registered for the envelope's oneof variant.
// Unknown variants and malformed messages are logged and dropped.
func (m *MessageMux) OnMessage(msg []byte) {
	var envelope clusterv1.ClusterMessage
	if err := proto.Unmarshal(msg, &envelope); err != nil {
		logger.Warn("Failed to unmarshal cluster message envelope", "error", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	switch v := envelope.Message.(type) {
	case *clusterv1.ClusterMessage_CacheInvalidation:
		if m.cacheInvalidationHandler != nil {
			m.cacheInvalidationHandler(v.CacheInvalidation)
		} else {
			logger.Warn("No handler registered for cache invalidation message")
		}
	default:
		logger.Warn("Unknown cluster message variant")
	}
}

// Wrap serializes a ClusterMessage envelope to bytes for broadcasting.
func Wrap(envelope *clusterv1.ClusterMessage) ([]byte, error) {
	return proto.Marshal(envelope)
}
