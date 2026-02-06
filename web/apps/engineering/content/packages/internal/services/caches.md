---
title: caches
description: "provides a centralized caching service for commonly accessed database entities"
---

Package caches provides a centralized caching service for commonly accessed database entities.

The caches package is designed to improve performance by reducing database load for frequently accessed data. It maintains in-memory caches with configurable freshness and staleness windows for various database entities used throughout the Unkey system.

Common use cases:

  - Looking up ratelimit namespaces by name
  - Retrieving ratelimit overrides that match specific criteria
  - Getting API keys by their hash
  - Retrieving permissions associated with a key ID

All caches are initialized with appropriate TTL settings and size limits, and include OpenTelemetry tracing for observability.

## Functions

### func DefaultFindFirstOp

```go
func DefaultFindFirstOp(err error) cache.Op
```

DefaultFindFirstOp returns the appropriate cache operation based on the sql error


## Types

### type Caches

```go
type Caches struct {
	// RatelimitNamespace caches ratelimit namespace lookups by name or ID.
	// Keys are cache.ScopedKey and values are db.FindRatelimitNamespace.
	RatelimitNamespace cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]

	// VerificationKeyByHash caches verification key lookups by their hash with pre-parsed data.
	// Keys are string (hash) and values are db.CachedKeyData (includes pre-parsed IP whitelist).
	VerificationKeyByHash cache.Cache[string, db.CachedKeyData]

	// LiveApiByID caches live API lookups by ID.
	// Keys are string (ID) and values are db.FindLiveApiByIDRow.
	LiveApiByID cache.Cache[cache.ScopedKey, db.FindLiveApiByIDRow]

	// Clickhouse Configuration caches clickhouse configuration lookups by workspace ID.
	// Keys are string (workspace ID) and values are db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow.
	ClickhouseSetting cache.Cache[string, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow]

	// KeyAuthToApiRow caches key_auth_id to api row mappings.
	// Keys are string (key_auth_id) and values are db.FindKeyAuthsByKeyAuthIdsRow (has both KeyAuthID and ApiID).
	KeyAuthToApiRow cache.Cache[cache.ScopedKey, db.FindKeyAuthsByKeyAuthIdsRow]

	// ApiToKeyAuthRow caches api_id to key_auth row mappings.
	// Keys are string (api_id) and values are db.FindKeyAuthsByIdsRow (has both KeyAuthID and ApiID).
	ApiToKeyAuthRow cache.Cache[cache.ScopedKey, db.FindKeyAuthsByIdsRow]

	// dispatcher handles routing of invalidation events to all caches in this process.
	// This is not exported as it's an internal implementation detail.
	dispatcher *clustering.InvalidationDispatcher
}
```

Caches holds all cache instances used throughout the application. Each field represents a specialized cache for a specific data entity.

#### func New

```go
func New(config Config) (Caches, error)
```

New creates and initializes all cache instances with appropriate settings.

It configures each cache with specific freshness/staleness windows, size limits, resource names for tracing, and wraps them with distributed invalidation if configured.

Parameters:

  - config: Configuration options including logger, clock, and optional topic for distributed invalidation.

Returns:

  - Caches: A struct containing all initialized cache instances.
  - error: An error if any cache failed to initialize.

All caches are thread-safe and can be accessed concurrently. If a CacheInvalidationTopic is provided, the caches will automatically handle distributed cache invalidation across cluster nodes when entries are modified.

Example:

	clock := clock.RealClock{}

	caches, err := caches.New(caches.Config{
	    Clock: clock,
	    CacheInvalidationTopic: topic, // optional for distributed invalidation
	})
	if err != nil {
	    log.Fatalf("Failed to initialize caches: %v", err)
	}

	// Use the caches - invalidation is automatic
	key, err := caches.KeyByHash.Get(ctx, "some-hash")

#### func (Caches) Close

```go
func (c *Caches) Close() error
```

Close shuts down the caches and cleans up resources.

### type Config

```go
type Config struct {
	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock

	// Topic for distributed cache invalidation
	CacheInvalidationTopic *eventstream.Topic[*cachev1.CacheInvalidationEvent]

	// NodeID identifies this node in the cluster (defaults to hostname-uniqueid to ensure uniqueness)
	NodeID string
}
```

Config defines the configuration options for initializing caches.

