package cluster

import (
	"sync"

	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
)

// MessageMux routes incoming cluster messages to registered handlers based on
// the envelope's type field. It sits between the cluster transport and
// application-level handlers, allowing multiple subsystems to share the same
// gossip cluster.
type MessageMux struct {
	mu       sync.RWMutex
	handlers map[string]func([]byte)
}

// NewMessageMux creates a new message multiplexer.
func NewMessageMux() *MessageMux {
	return &MessageMux{
		mu:       sync.RWMutex{},
		handlers: make(map[string]func([]byte)),
	}
}

// Handle registers a handler for the given message type.
// Only one handler per type is supported; a second call with the same type
// replaces the previous handler.
func (m *MessageMux) Handle(msgType string, handler func([]byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[msgType] = handler
}

// OnMessage deserializes the ClusterMessage envelope and dispatches the inner
// payload to the handler registered for the envelope's type.
// Unknown types and malformed messages are logged and dropped.
func (m *MessageMux) OnMessage(msg []byte) {
	var envelope clusterv1.ClusterMessage
	if err := proto.Unmarshal(msg, &envelope); err != nil {
		logger.Warn("Failed to unmarshal cluster message envelope", "error", err)
		return
	}

	m.mu.RLock()
	handler, ok := m.handlers[envelope.Type]
	m.mu.RUnlock()

	if !ok {
		logger.Warn("No handler registered for cluster message type", "type", envelope.Type)
		return
	}

	handler(envelope.Payload)
}

// Wrap serializes an application payload into a ClusterMessage envelope.
func Wrap(msgType string, payload []byte) ([]byte, error) {
	envelope := &clusterv1.ClusterMessage{
		Type:    msgType,
		Payload: payload,
	}
	return proto.Marshal(envelope)
}
