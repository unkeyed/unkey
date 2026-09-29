package caches

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
)

// cacheWithLinkedRemoval evicts matching keys from a related cache while
// preserving the primary cache's value type.
type cacheWithLinkedRemoval[V, LinkedV any] struct {
	cache.Cache[string, V]
	linked cache.Cache[string, LinkedV]
}

// Remove evicts keys from both caches.
func (c *cacheWithLinkedRemoval[V, LinkedV]) Remove(ctx context.Context, keys ...string) {
	c.Cache.Remove(ctx, keys...)
	c.linked.Remove(ctx, keys...)
}

// Clear removes every entry from both caches.
func (c *cacheWithLinkedRemoval[V, LinkedV]) Clear(ctx context.Context) {
	c.Cache.Clear(ctx)
	c.linked.Clear(ctx)
}
