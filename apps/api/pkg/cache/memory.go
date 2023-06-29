package cache

import (
	"context"
	"github.com/chronark/unkey/apps/api/pkg/tracing"
	"sync"
	"time"
)

type Cache[T any] interface {
	// Get returns the value for the given key.
	// If the key is not found, found will be false.
	Get(ctx context.Context, key string) (value T, found bool)
	// Sets the value for the given key.
	Set(ctx context.Context, key string, value T)
	// Removes the key from the cache.
	Remove(ctx context.Context, key string)
	// Returns true if the key is found.
	Contains(ctx context.Context, key string) bool
	// Returns the number of items in the cache.
	Size() int
	// Removes all items from the cache.
	Clear()

	// Remove all expired items
	DeleteExpired()
}

type entry[T any] struct {
	value T
	exp   time.Time
}

func (e *entry[T]) expired() bool {
	if e.exp.IsZero() {
		return false
	}
	return e.exp.Before(time.Now())
}

type inMemoryCache[T any] struct {
	sync.RWMutex
	items  map[string]entry[T]
	ttl    time.Duration
	tracer tracing.Tracer
}

type Config struct {
	Ttl    time.Duration
	Tracer tracing.Tracer
}

func NewInMemoryCache[T any](config Config) Cache[T] {
	c := &inMemoryCache[T]{
		items:  make(map[string]entry[T]),
		ttl:    config.Ttl,
		tracer: config.Tracer,
	}

	if c.ttl > 0 {
		go func() {

			for range time.NewTicker(c.ttl).C {
				c.Lock()
				for key, val := range c.items {
					if val.expired() {
						delete(c.items, key)
					}
				}
				c.Unlock()
			}

		}()
	}

	return c
}

func (c *inMemoryCache[T]) DeleteExpired() {
	c.Lock()
	defer c.Unlock()
	for key, entry := range c.items {
		if entry.expired() {
			delete(c.items, key)
		}
	}
}
func (c *inMemoryCache[T]) Get(ctx context.Context, key string) (T, bool) {
	ctx, span := c.tracer.Start(ctx, "cache.get")
	defer span.End()
	c.RLock()
	entry, ok := c.items[key]
	c.RUnlock()

	if !ok {
		// This hack is necessary because you can not return nil as T
		var t T
		return t, false
	}
	if entry.expired() {
		c.Lock()
		delete(c.items, key)
		c.Unlock()
		// This hack is necessary because you can not return nil as T
		var t T
		return t, false
	}

	return entry.value, true
}

func (c *inMemoryCache[T]) Set(ctx context.Context, key string, value T) {
	ctx, span := c.tracer.Start(ctx, "cache.set")
	defer span.End()
	c.Lock()
	defer c.Unlock()

	c.items[key] = entry[T]{
		value: value,
		exp:   time.Now().Add(c.ttl),
	}

}

func (c *inMemoryCache[T]) Remove(ctx context.Context, key string) {
	ctx, span := c.tracer.Start(ctx, "cache.remove")
	defer span.End()
	c.Lock()
	defer c.Unlock()

	delete(c.items, key)

}

func (c *inMemoryCache[T]) Contains(ctx context.Context, key string) bool {
	ctx, span := c.tracer.Start(ctx, "cache.contains")
	defer span.End()
	c.RLock()

	entry, ok := c.items[key]
	c.RUnlock()
	if !ok {
		return false
	}
	if entry.exp.Before(time.Now()) {
		c.Lock()
		delete(c.items, key)
		c.Unlock()
		return false
	}

	return true
}

func (c *inMemoryCache[T]) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.items)
}

func (c *inMemoryCache[T]) Clear() {
	c.Lock()
	defer c.Unlock()
	for key := range c.items {
		delete(c.items, key)
	}

}
