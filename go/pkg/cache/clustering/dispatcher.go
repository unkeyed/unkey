package clustering

import (
	"context"
	"sync"

	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
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
	logger   logging.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// noopDispatcher is a no-op implementation that does nothing.
// Used when clustering is disabled.
type noopDispatcher struct{}

func (n *noopDispatcher) Register(handler InvalidationHandler) {}
func (n *noopDispatcher) Close() error                         { return nil }
func (n *noopDispatcher) handleEvent(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
	return nil
}

// NewNoopDispatcher creates a dispatcher that does nothing.
// Use this when clustering is disabled.
func NewNoopDispatcher() *InvalidationDispatcher {
	return &InvalidationDispatcher{
		handlers: make(map[string]InvalidationHandler),
		logger:   logging.NewNoop(),
		ctx:      context.Background(),
		cancel:   func() {},
	}
}

// NewInvalidationDispatcher creates a new dispatcher that routes invalidation
// events to registered caches.
//
// Panics if topic or logger is nil - use NewNoopDispatcher() if clustering is disabled.
func NewInvalidationDispatcher(topic *eventstream.Topic[*cachev1.CacheInvalidationEvent], logger logging.Logger) *InvalidationDispatcher {
	if topic == nil {
		panic("topic is required for InvalidationDispatcher - use NewNoopDispatcher() if clustering is disabled")
	}
	if logger == nil {
		panic("logger is required for InvalidationDispatcher")
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &InvalidationDispatcher{
		handlers: make(map[string]InvalidationHandler),
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}

	d.consumer = topic.NewConsumer()
	d.consumer.Consume(ctx, d.handleEvent)

	return d
}

// handleEvent processes a single invalidation event by routing it to
// the appropriate cache handler.
func (d *InvalidationDispatcher) handleEvent(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
	d.mu.RLock()
	handler, exists := d.handlers[event.CacheName]
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
	d.cancel()

	if d.consumer != nil {
		return d.consumer.Close()
	}

	return nil
}
