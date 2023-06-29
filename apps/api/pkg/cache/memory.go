package cache

import (
	"sync"
	"time"
)

type Cache[T any] interface {
	// Get returns the value for the given key.
	// If the key is not found, found will be false.
	Get(key string) (value T, found bool)
	// Sets the value for the given key.
	Set(key string, value T)
	// Removes the key from the cache.
	Remove(key string)
	// Returns true if the key is found.
	Contains(key string) bool
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
	items map[string]entry[T]
	ttl   time.Duration
}

func NewInMemoryCache[T any](ttl time.Duration) Cache[T] {
	c := &inMemoryCache[T]{
		items: make(map[string]entry[T]),
		ttl:   ttl,
	}

	if ttl > 0 {
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
func (c *inMemoryCache[T]) Get(key string) (T, bool) {
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

func (c *inMemoryCache[T]) Set(key string, value T) {
	c.Lock()
	defer c.Unlock()

	c.items[key] = entry[T]{
		value: value,
		exp:   time.Now().Add(c.ttl),
	}

}

func (c *inMemoryCache[T]) Remove(key string) {
	c.Lock()
	defer c.Unlock()

	delete(c.items, key)

}

func (c *inMemoryCache[T]) Contains(key string) bool {
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
