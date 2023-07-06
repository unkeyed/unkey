package cache

import (
	"context"
)

type Cache[T any] interface {
	// Get returns the value for the given key.
	// If the key is not found, found will be false.
	Get(ctx context.Context, key string) (value T, found bool)

	// Sets the value for the given key.
	Set(ctx context.Context, key string, value T)
	// Removes the key from the cache.
	Remove(ctx context.Context, key string)
}
