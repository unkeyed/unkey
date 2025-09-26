package clustering

import (
	"context"
	"sync"

	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// InvalidationProcessor is an interface that cluster caches implement
type InvalidationProcessor interface {
	ProcessInvalidationEvent(ctx context.Context, event *cachev1.CacheInvalidationEvent) bool
	Name() string
}

// Manager handles invalidation events for all cluster caches
type Manager struct {
	mu         sync.RWMutex
	processors map[string]InvalidationProcessor // keyed by cache name
	consumer   eventstream.Consumer[*cachev1.CacheInvalidationEvent]
}

// globalManager is the singleton instance
var (
	globalManager *Manager
	once          sync.Once
)

// GetManager returns the global invalidation manager
func GetManager() *Manager {
	once.Do(func() {
		globalManager = &Manager{
			processors: make(map[string]InvalidationProcessor),
		}
	})

	return globalManager
}

// Register adds a cache to receive invalidation events
func (m *Manager) Register(processor InvalidationProcessor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processors[processor.Name()] = processor
}

// Start begins consuming invalidation events
func (m *Manager) Start(topic *eventstream.Topic[*cachev1.CacheInvalidationEvent], logger logging.Logger) {
	if topic == nil || m.consumer != nil {
		return
	}

	m.consumer = topic.NewConsumer()
	m.consumer.Consume(context.Background(), func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		m.mu.RLock()
		processor, exists := m.processors[event.CacheName]
		m.mu.RUnlock()

		// If we don't have a processor for this cache, return early
		if !exists {
			return nil
		}

		processor.ProcessInvalidationEvent(ctx, event)
		return nil
	})
}
