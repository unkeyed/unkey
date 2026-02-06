package clustering

import (
	"context"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
)

// Broadcaster defines the interface for broadcasting cache invalidation events
// across cluster nodes. Implementations handle serialization and transport.
type Broadcaster interface {
	// Broadcast sends one or more cache invalidation events to other nodes.
	Broadcast(ctx context.Context, events ...*cachev1.CacheInvalidationEvent) error

	// Subscribe registers a handler for incoming invalidation events from other nodes.
	Subscribe(ctx context.Context, handler func(context.Context, *cachev1.CacheInvalidationEvent) error)

	// Close shuts down the broadcaster and releases resources.
	Close() error
}
