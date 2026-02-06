package clustering

import (
	"context"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
)

// noopBroadcaster is a no-op implementation of Broadcaster.
// Used when clustering is disabled.
type noopBroadcaster struct{}

var _ Broadcaster = (*noopBroadcaster)(nil)

// NewNoopBroadcaster returns a Broadcaster that does nothing.
func NewNoopBroadcaster() Broadcaster {
	return &noopBroadcaster{}
}

func (b *noopBroadcaster) Broadcast(_ context.Context, _ ...*cachev1.CacheInvalidationEvent) error {
	return nil
}

func (b *noopBroadcaster) Subscribe(_ context.Context, _ func(context.Context, *cachev1.CacheInvalidationEvent) error) {
}

func (b *noopBroadcaster) Close() error {
	return nil
}
