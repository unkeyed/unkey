package bus

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// noopBus is the Bus implementation used when gossip is disabled (dev,
// single-node deploys, tests). All operations are safe to call but do
// nothing.
type noopBus struct{}

var _ Bus = noopBus{}

func (noopBus) Publish(context.Context, string, proto.Message) error { return nil }

// Subscribe returns an unsubscribe that is also a no-op. Callers can defer
// the result without checking which Bus implementation they hold.
func (noopBus) Subscribe(string, Handler) func() {
	return func() {}
}

// Query returns an immediately-closed channel: a noop bus has no peers to
// reply, and callers that range over the channel exit cleanly.
func (noopBus) Query(context.Context, string, proto.Message) (<-chan QueryResponse, error) {
	ch := make(chan QueryResponse)
	close(ch)
	return ch, nil
}

func (noopBus) Members() []Member { return nil }
func (noopBus) Pause()            {}
func (noopBus) Resume()           {}
func (noopBus) Close() error      { return nil }

// NewNoop returns a Bus that drops all writes and reports an empty cluster.
// Use it when gossip is not configured for the running process.
func NewNoop() Bus {
	return noopBus{}
}
