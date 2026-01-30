package healthcheck

import "context"

// Noop is a no-op heartbeat implementation for testing or when heartbeats are disabled.
type Noop struct{}

// NewNoop creates a new no-op heartbeat that does nothing.
func NewNoop() *Noop {
	return &Noop{}
}

// Ping does nothing and always returns nil.
func (*Noop) Ping(context.Context) error {
	return nil
}
