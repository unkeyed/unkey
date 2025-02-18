package cache

import (
	"context"
)

type Cache[K comparable, V any] interface {
	// Get returns the value for the given key.
	// If the key is not found, found will be false.
	Get(ctx context.Context, key K) (value V, hit CacheHit)

	// Sets the value for the given key.
	Set(ctx context.Context, key K, value V)

	// Sets the given key to null, indicating that the value does not exist in the origin.
	SetNull(ctx context.Context, key K)

	// Removes the key from the cache.
	Remove(ctx context.Context, key K)

	SWR(ctx context.Context, key K, refreshFromOrigin func(ctx context.Context) (V, error), translateError func(error) CacheHit) (value V, err error)

	// Dump returns a serialized representation of the cache.
	Dump(ctx context.Context) ([]byte, error)

	// Restore restores the cache from a serialized representation.
	Restore(ctx context.Context, data []byte) error

	// Clear removes all entries from the cache.
	Clear(ctx context.Context)
}

type Key interface {
	ToString() string
}

type CacheHit int

const (
	Null CacheHit = iota
	// The entry was in the cache and can be used
	Hit
	// The entry was not in the cache
	Miss
	// The entry did not exist in the origin

)
