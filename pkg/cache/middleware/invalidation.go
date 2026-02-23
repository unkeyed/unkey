package middleware

import (
	"context"
	"fmt"
	"sync"

	"github.com/unkeyed/unkey/pkg/cache"
)

// InvalidationRegistry maps cache resource names to functions that can remove
// entries by their string-encoded keys. Built once during cache construction;
// used by the API's cache invalidation endpoint.
type InvalidationRegistry struct {
	mu       sync.RWMutex
	handlers map[string]func(ctx context.Context, keys []string) error
}

// NewInvalidationRegistry creates an empty registry.
func NewInvalidationRegistry() *InvalidationRegistry {
	return &InvalidationRegistry{
		mu:       sync.RWMutex{},
		handlers: make(map[string]func(ctx context.Context, keys []string) error),
	}
}

// register adds an invalidation handler for the given cache name.
func (r *InvalidationRegistry) register(name string, fn func(ctx context.Context, keys []string) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = fn
}

// Invalidate removes the given keys from the named cache.
// Returns an error if the cache name is not registered.
func (r *InvalidationRegistry) Invalidate(ctx context.Context, cacheName string, keys []string) error {
	r.mu.RLock()
	fn, ok := r.handlers[cacheName]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown cache name: %s", cacheName)
	}
	return fn(ctx, keys)
}

// WithInvalidation registers a cache for string-key-based invalidation
// and returns the cache unchanged. This is not a wrapping middleware —
// it only registers a side-channel so the cache can be invalidated by
// name through the registry.
//
// The cache name is taken from c.Name() (the Resource field set in cache.Config).
//
// parseKey converts the raw string key (from the HTTP request) into the
// cache's typed key K. For string-keyed caches, use StringKeyParser.
// For ScopedKey caches, use cache.ParseScopedKey.
func WithInvalidation[K comparable, V any](
	c cache.Cache[K, V],
	registry *InvalidationRegistry,
	parseKey func(string) (K, error),
) cache.Cache[K, V] {
	registry.register(c.Name(), func(ctx context.Context, keys []string) error {
		parsed := make([]K, len(keys))
		for i, s := range keys {
			k, err := parseKey(s)
			if err != nil {
				return err
			}
			parsed[i] = k
		}
		c.Remove(ctx, parsed...)
		return nil
	})
	return c
}

// StringKeyParser is a key parser for caches keyed by plain strings.
func StringKeyParser(s string) (string, error) {
	return s, nil
}
