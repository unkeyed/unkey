package clustering

import (
	"context"
	"sync"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/eventstream"
)

// InvalidationHandler is an interface that cluster caches implement
// to handle cache invalidation events.
type InvalidationHandler interface {
	HandleInvalidation(ctx context.Context, event *cachev1.CacheInvalidationEvent) bool
	Name() string
}

// InvalidationDispatcher routes cache invalidation events from Kafka
// to the appropriate cache instances within a single process.
//
// In a distributed system, each process (server) has one dispatcher
// that consumes invalidation events and routes them to all local caches
// based on the cache name in the event.
type InvalidationDispatcher struct {
	mu       sync.RWMutex
	handlers map[string]InvalidationHandler // keyed by cache name
	consumer eventstream.Consumer[*cachev1.CacheInvalidationEvent]
}

// NewInvalidationDispatcher creates a new dispatcher that routes invalidation
// events to registered caches.
//
// Returns an error if topic is nil - use NewNoopDispatcher() if clustering is disabled.
func NewInvalidationDispatcher(topic *eventstream.Topic[*cachev1.CacheInvalidationEvent]) (*InvalidationDispatcher, error) {
	err := assert.All(
		assert.NotNil(topic, "topic is required for InvalidationDispatcher - use NewNoopDispatcher() if clustering is disabled"),
	)
	if err != nil {
		return nil, err
	}

	d := &InvalidationDispatcher{
		mu:       sync.RWMutex{},
		consumer: nil,
		handlers: make(map[string]InvalidationHandler),
	}

	d.consumer = topic.NewConsumer()
	d.consumer.Consume(context.Background(), d.handleEvent)

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
	if d.consumer != nil {
		return d.consumer.Close()
	}
	return nil
}
