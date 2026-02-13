package cluster

import (
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
)

// MessageMux fans out incoming cluster messages to all registered subscribers.
// It sits between the cluster transport and application-level handlers, allowing
// multiple subsystems to share the same gossip cluster.
type MessageMux struct {
	handlers []func(*clusterv1.ClusterMessage)
}

// NewMessageMux creates a new message multiplexer.
func NewMessageMux() *MessageMux {
	return &MessageMux{
		handlers: nil,
	}
}

// Subscribe registers a handler that will receive all cluster messages.
// Handlers are responsible for filtering by message type (e.g. checking
// the oneof variant).
func (m *MessageMux) Subscribe(handler func(*clusterv1.ClusterMessage)) {
	m.handlers = append(m.handlers, handler)
}

// OnMessage dispatches a ClusterMessage to all registered subscribers.
func (m *MessageMux) OnMessage(msg *clusterv1.ClusterMessage) {
	for _, h := range m.handlers {
		h(msg)
	}
}
