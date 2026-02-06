---
title: cache
description: "provides a generic, thread-safe caching system with support"
---

Package cache provides a generic, thread-safe caching system with support for time-based expiration, custom eviction policies, and observability.

The cache implementation uses a combination of LRU (Least Recently Used) and TTL (Time To Live) strategies to manage memory efficiently. It supports stale-while-revalidate (SWR) behavior, allowing expired items to be served while being refreshed asynchronously in the background.

Basic usage:

	// Create a new cache with default settings
	c := cache.New[string, User](cache.Config[string, User]{
	    Fresh:    time.Minute,    // Items considered fresh for 1 minute
	    Stale:    time.Hour,      // Items can be served stale for up to 1 hour
	    MaxSize:  10000,          // Store up to 10,000 items
	    Resource: "users",        // For metrics and logging
	})

	// Store an item
	c.Set(ctx, "user:123", user)

	// Retrieve an item
	user, hit := c.Get(ctx, "user:123")
	if hit == cache.Hit {
	    // Use the cached user
	} else {
	    // Cache miss, fetch from database
	}

	// SWR pattern
	user, err := c.SWR(ctx, "user:123",
	    func(ctx context.Context) (User, error) {
	        // This will only be called if the cache doesn't have a fresh value
	        return fetchUserFromDatabase(ctx, "123")
	    },
	    func(err error) cache.CacheHit {
	        // Translate errors to cache behavior
	        if errors.Is(err, sql.ErrNoRows) {
	            return cache.Null  // Mark as explicitly not found
	        }
	        return cache.Miss      // Mark as a transient error
	    },
	)

## Variables

```go
var ScopedKeyFromString = func(s string) (ScopedKey, error) { return ParseScopedKey(s) }
```

```go
var ScopedKeyToString = func(k ScopedKey) string { return k.String() }
```


## Types

### type Cache

```go
type Cache[K comparable, V any] interface {
	// Get returns the value for the given key.
	// If the key is not found, found will be false.
	Get(ctx context.Context, key K) (value V, hit CacheHit)

	// GetMany returns values for multiple keys.
	// Returns maps of values and cache hits indexed by key.
	GetMany(ctx context.Context, keys []K) (values map[K]V, hits map[K]CacheHit)

	// Sets the value for the given key.
	Set(ctx context.Context, key K, value V)

	// SetMany sets multiple key-value pairs.
	SetMany(ctx context.Context, values map[K]V)

	// Sets the given key to null, indicating that the value does not exist in the origin.
	SetNull(ctx context.Context, key K)

	// SetNullMany sets multiple keys to null.
	SetNullMany(ctx context.Context, keys []K)

	// Remove removes one or more keys from the cache.
	// Multiple keys can be provided for efficient bulk removal.
	Remove(ctx context.Context, keys ...K)

	// SWR performs stale-while-revalidate: returns cached data immediately while
	// optionally refreshing in the background. The op function determines whether
	// to write the refreshed value, write null, or take no action based on the refresh error.
	SWR(ctx context.Context, key K, refreshFromOrigin func(ctx context.Context) (V, error), op func(error) Op) (value V, hit CacheHit, err error)

	// SWRMany performs stale-while-revalidate for multiple keys.
	// refreshFromOrigin receives keys that need to be fetched and returns a map of values.
	SWRMany(ctx context.Context, keys []K, refreshFromOrigin func(ctx context.Context, keys []K) (map[K]V, error), op func(error) Op) (values map[K]V, hits map[K]CacheHit, err error)

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
```

Cache defines the interface for a generic caching layer that supports typed keys and values. Implementations must be safe for concurrent use and handle cache misses gracefully. The interface supports both single and batch operations, as well as stale-while-revalidate patterns for background refresh of cached data.

#### func New

```go
func New[K comparable, V any](config Config[K, V]) (Cache[K, V], error)
```

New creates a new cache instance

#### func NewNoopCache

```go
func NewNoopCache[K comparable, V any]() Cache[K, V]
```

### type CacheHit

```go
type CacheHit int
```

```go
const (
	// Null indicates the entry exists but has a null value
	Null CacheHit = iota
	// Hit indicates the entry was in the cache and can be used
	Hit
	// Miss indicates the entry was not in the cache
	Miss
)
```

### type Config

```go
type Config[K comparable, V any] struct {
	// How long the data is considered fresh
	// Subsequent requests in this time will try to use the cache
	Fresh time.Duration

	// Subsequent requests that are not fresh but within the stale time will return cached data but also trigger
	// fetching from the origin server
	Stale time.Duration

	// Start evicting the least recently used entry when the cache grows to MaxSize
	MaxSize int

	Resource string

	Clock clock.Clock
}
```

### type Key

```go
type Key interface {
	ToString() string
}
```

Key represents a cache key that can be serialized to a string representation. Implementations should ensure ToString returns a unique, stable string for each distinct key.

### type Middleware

```go
type Middleware[K comparable, V any] func(Cache[K, V]) Cache[K, V]
```

### type Op

```go
type Op int
```

```go
const (
	// do nothing
	Noop Op = iota
	// The entry was in the cache and should be stored in the cache
	WriteValue Op = iota
	// The entry was not found in the origin, we must store that information
	// in the cache
	WriteNull Op = iota
)
```

### type ScopedKey

```go
type ScopedKey struct {
	// WorkspaceID is the unique identifier for the workspace that owns this resource.
	// This ensures that cache keys are isolated between different workspaces,
	// preventing accidental data leakage or cache collisions.
	WorkspaceID string

	// Key is the identifier for the resource within the workspace.
	// This can be a user-provided name, system-generated ID, slug, or any other
	// string identifier that uniquely identifies the resource within the workspace.
	//
	// The key is only guaranteed to be unique within the workspace context.
	// Different workspaces may have resources with the same key value.
	Key string
}
```

ScopedKey represents a cache key that is scoped to a specific workspace.

This type is designed for caching data where keys are only unique within a workspace context, rather than being globally unique. For example, a user might create a ratelimit namespace called "api-calls" which is unique within their workspace but could exist in multiple workspaces.

The ScopedKey ensures cache isolation between workspaces by combining the workspace ID with the resource key, preventing cache collisions and data leakage between different workspaces.

### Usage

Use ScopedKey when caching data that is workspace-specific:

	// Cache a ratelimit namespace by name
	key := cache.ScopedKey{
		WorkspaceID: "ws_123",
		Key:         "api-calls",
	}

	// Cache by ID (still workspace-scoped for consistency)
	key := cache.ScopedKey{
		WorkspaceID: "ws_123",
		Key:         "ns_456",
	}

	// Cache any workspace-scoped resource
	key := cache.ScopedKey{
		WorkspaceID: "ws_123",
		Key:         "some-resource-identifier",
	}

### Design Rationale

We chose this approach over concatenating strings because it provides type safety and makes the workspace scoping explicit in the API. It also allows for future extension if additional scoping dimensions are needed.

The generic Key field can hold any string identifier (names, IDs, slugs, etc.) while maintaining consistent workspace isolation across all cache usage patterns.

#### func ParseScopedKey

```go
func ParseScopedKey(s string) (ScopedKey, error)
```

ParseScopedKey parses a string in the format "workspace\_id:key" into a ScopedKey. Returns an error if the string is not in the expected format.

#### func (ScopedKey) String

```go
func (k ScopedKey) String() string
```

