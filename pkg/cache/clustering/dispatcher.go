package clustering

import (
	"context"
	"sync"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/pkg/assert"
)

// InvalidationHandler is an interface that cluster caches implement
// to handle cache invalidation events.
type InvalidationHandler interface {
	HandleInvalidation(ctx context.Context, event *cachev1.CacheInvalidationEvent) bool
	Name() string
}

// InvalidationDispatcher routes cache invalidation events from the
// broadcaster to the appropriate cache instances within a single process.
//
// In a distributed system, each process (server) has one dispatcher
// that receives invalidation events and routes them to all local caches
// based on the cache name in the event.
type InvalidationDispatcher struct {
	mu          sync.RWMutex
	handlers    map[string]InvalidationHandler // keyed by cache name
	broadcaster Broadcaster
}

// NewInvalidationDispatcher creates a new dispatcher that routes invalidation
// events to registered caches.
//
// Returns an error if broadcaster is nil - use NewNoopDispatcher() if clustering is disabled.
func NewInvalidationDispatcher(broadcaster Broadcaster) (*InvalidationDispatcher, error) {
	err := assert.All(
		assert.NotNil(broadcaster, "broadcaster is required for InvalidationDispatcher - use NewNoopDispatcher() if clustering is disabled"),
	)
	if err != nil {
		return nil, err
	}

	d := &InvalidationDispatcher{
		mu:          sync.RWMutex{},
		handlers:    make(map[string]InvalidationHandler),
		broadcaster: broadcaster,
	}

	broadcaster.Subscribe(context.Background(), d.handleEvent)

	return d, nil
}

// handleEvent processes a single invalidation event by routing it to
// the appropriate cache handler.
func (d *InvalidationDispatcher) handleEvent(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
	d.mu.RLock()
	handler, exists := d.handlers[event.GetCacheName()]
	d.mu.RUnlock()

	// If we don't have a handler for this cache, skip it
	if !exists {
		return nil
	}

	handler.HandleInvalidation(ctx, event)
	return nil
}

// Register adds a cache to receive invalidation events.
// The cache will receive events matching its cache name.
func (d *InvalidationDispatcher) Register(handler InvalidationHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[handler.Name()] = handler
}

// Close stops the dispatcher and cleans up resources.
func (d *InvalidationDispatcher) Close() error {
	if d.broadcaster != nil {
		return d.broadcaster.Close()
	}
	return nil
}
