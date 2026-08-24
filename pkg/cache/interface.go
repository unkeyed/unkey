package cache

import (
	"context"
)

// Cache defines the interface for a generic caching layer that supports typed keys and values.
// Implementations must be safe for concurrent use and handle cache misses gracefully.
// The interface supports stale-while-revalidate patterns for background refresh
// of cached data.
type Cache[K comparable, V any] interface {
	// Get returns a fresh or stale value for the given key without revalidation.
	// If the key is not found or expired, hit will be Miss.
	Get(ctx context.Context, key K) (value V, hit CacheHit)

	// Sets the value for the given key.
	Set(ctx context.Context, key K, value V)

	// Sets the given key to null, indicating that the value does not exist in the origin.
	SetNull(ctx context.Context, key K)

	// Remove removes one or more keys from the cache.
	// Multiple keys can be provided for efficient bulk removal.
	Remove(ctx context.Context, keys ...K)

	// SWR performs stale-while-revalidate: returns cached data immediately while
	// optionally refreshing in the background. The op function determines whether
	// to write the refreshed value, write null, or take no action based on the refresh error.
	SWR(ctx context.Context, key K, refreshFromOrigin func(ctx context.Context) (V, error), op func(error) Op) (value V, hit CacheHit, err error)

	// SWRWithFallback checks multiple candidate keys in order, returning the first hit.
	// On miss, calls refreshFromOrigin which returns the value AND the canonical key to cache under.
	// This is useful for wildcard/fallback patterns where multiple lookups share a single cached value.
	// Example: domains foo.example.com and bar.example.com both use wildcard cert *.example.com
	SWRWithFallback(ctx context.Context, candidates []K, refreshFromOrigin func(ctx context.Context) (value V, canonicalKey K, err error), op func(error) Op) (value V, hit CacheHit, err error)

	// Dump returns a serialized representation of the cache.
	Dump(ctx context.Context) ([]byte, error)

	// Restore restores the cache from a serialized representation.
	Restore(ctx context.Context, data []byte) error

	// Clear removes all entries from the cache.
	Clear(ctx context.Context)

	// Name returns the name of this cache instance.
	Name() string
}

// Key represents a cache key that can be serialized to a string representation.
// Implementations should ensure ToString returns a unique, stable string for each distinct key.
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
