package cache

import (
	"context"
	"sync"
	"time"
)

type Cache[T any] interface {
	// Get returns the value for the given key.
	// If the key is not found, found will be false.
	Get(ctx context.Context, key string, refresh bool) (value T, found bool)

	// Sets the value for the given key.
	Set(ctx context.Context, key string, value T, exp time.Time)
	// Removes the key from the cache.
	Remove(ctx context.Context, key string)
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
	items map[string]entry[T]
}

func NewInMemoryCache[T any]() Cache[T] {
	c := &inMemoryCache[T]{
		items: make(map[string]entry[T]),
	}

	go func() {

		for range time.NewTicker(time.Minute).C {
			c.Lock()
			for key, val := range c.items {
				if val.expired() {
					delete(c.items, key)
				}
			}
			c.Unlock()
		}

	}()

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
func (c *inMemoryCache[T]) Get(ctx context.Context, key string, refresh bool) (T, bool) {

	c.RLock()
	e, ok := c.items[key]
	c.RUnlock()

	if !ok {
		// This hack is necessary because you can not return nil as T
		var t T
		return t, false
	}

	if e.expired() {
		c.Lock()
		defer c.Unlock()
		delete(c.items, key)
		// This hack is necessary because you can not return nil as T
		var t T
		return t, false
	}

	return e.value, true
}

func (c *inMemoryCache[T]) Set(ctx context.Context, key string, value T, exp time.Time) {

	c.Lock()
	defer c.Unlock()

	c.items[key] = entry[T]{
		value: value,
		exp:   exp,
	}

}

func (c *inMemoryCache[T]) Remove(ctx context.Context, key string) {

	c.Lock()
	defer c.Unlock()

	delete(c.items, key)

}
