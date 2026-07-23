package healthcheck

import (
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
)

// Noop is a no-op heartbeat implementation for testing or when heartbeats are disabled.
type Noop struct{}

// NewNoop creates a new no-op heartbeat that does nothing.
func NewNoop() *Noop {
	return &Noop{}
}

// Ping logs that the heartbeat is disabled and returns nil.
func (*Noop) Ping(context.Context) error {
	logger.Warn("heartbeat not sent (noop heartbeat)")
	return nil
}
