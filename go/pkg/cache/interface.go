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

	// Remove removes one or more keys from the cache.
	// Multiple keys can be provided for efficient bulk removal.
	Remove(ctx context.Context, keys ...K)

	SWR(ctx context.Context, key K, refreshFromOrigin func(ctx context.Context) (V, error), op func(error) Op) (value V, hit CacheHit, err error)

	// Dump returns a serialized representation of the cache.
	Dump(ctx context.Context) ([]byte, error)

	// Restore restores the cache from a serialized representation.
	Restore(ctx context.Context, data []byte) error

	// Clear removes all entries from the cache.
	Clear(ctx context.Context)

	// Name returns the name of this cache instance.
	Name() string
}

type Key interface {
	ToString() string
}

type CacheHit int

const (
	// Null indicates the entry exists but has a null value
	Null CacheHit = iota
	// Hit indicates the entry was in the cache and can be used
	Hit
	// Miss indicates the entry was not in the cache
	Miss
)

type Op int

const (
	// do nothing
	Noop Op = iota
	// The entry was in the cache and should be stored in the cache
	WriteValue Op = iota
	// The entry was not found in the origin, we must store that information
	// in the cache
	WriteNull Op = iota
)
