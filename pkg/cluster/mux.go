package cluster

import (
	"fmt"
	"sync"
	"time"

	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// MessageMux fans out incoming cluster messages to all registered subscribers.
// It sits between the cluster transport and application-level handlers, allowing
// multiple subsystems to share the same gossip cluster.
type MessageMux struct {
	mu       sync.RWMutex
	handlers []func(*clusterv1.ClusterMessage)
}

// NewMessageMux creates a new message multiplexer.
func NewMessageMux() *MessageMux {
	return &MessageMux{
		handlers: nil,
	}
}

// subscribe adds a raw handler that receives all cluster messages.
func (m *MessageMux) subscribe(handler func(*clusterv1.ClusterMessage)) {
	m.mu.Lock()
	m.handlers = append(m.handlers, handler)
	m.mu.Unlock()
}

// Subscribe registers a typed handler that only receives messages matching
// the given oneof payload variant. The type assertion is handled automatically.
func Subscribe[T clusterv1.IsClusterMessage_Payload](mux *MessageMux, handler func(T)) {
	mux.subscribe(func(msg *clusterv1.ClusterMessage) {
		payload, ok := msg.Payload.(T)
		if !ok {
			return
		}

		handler(payload)
	})
}

// OnMessage dispatches a ClusterMessage to all registered subscribers.
func (m *MessageMux) OnMessage(msg *clusterv1.ClusterMessage) {
	now := time.Now().UnixMilli()
	latencyMs := now - msg.SentAtMs

	logger.Info("cluster message received",
		"latency_ms", latencyMs,
		"received_at_ms", now,
		"sent_at_ms", msg.SentAtMs,
		"source_region", msg.SourceRegion,
		"sender_node", msg.SenderNode,
		"direction", msg.Direction.String(),
		"payload_type", fmt.Sprintf("%T", msg.Payload),
	)

	m.mu.RLock()
	snapshot := make([]func(*clusterv1.ClusterMessage), len(m.handlers))
	copy(snapshot, m.handlers)
	m.mu.RUnlock()

	for _, h := range snapshot {
		h(msg)
	}
}
