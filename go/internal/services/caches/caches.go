package caches

import (
	"time"

	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/clustering"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Caches holds all cache instances used throughout the application.
// Each field represents a specialized cache for a specific data entity.
type Caches struct {
	// RatelimitNamespace caches ratelimit namespace lookups by name or ID.
	// Keys are cache.ScopedKey and values are db.FindRatelimitNamespace.
	RatelimitNamespace cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]

	// VerificationKeyByHash caches verification key lookups by their hash.
	// Keys are string (hash) and values are db.VerificationKey.
	VerificationKeyByHash cache.Cache[string, db.FindKeyForVerificationRow]

	// LiveApiByID caches live API lookups by ID.
	// Keys are string (ID) and values are db.FindLiveApiByIDRow.
	LiveApiByID cache.Cache[cache.ScopedKey, db.FindLiveApiByIDRow]

	// Clickhouse Configuration caches clickhouse configuration lookups by workspace ID.
	// Keys are string (workspace ID) and values are db.ClickhouseWorkspaceSetting.
	ClickhouseSetting cache.Cache[string, db.ClickhouseWorkspaceSetting]

	// KeyAuthToApiRow caches key_auth_id to api row mappings.
	// Keys are string (key_auth_id) and values are db.FindKeyAuthsByIdsRow (has both KeyAuthID and ApiID).
	KeyAuthToApiRow cache.Cache[cache.ScopedKey, db.FindKeyAuthsByIdsRow]

	// ApiToKeyAuthRow caches api_id to key_auth row mappings.
	// Keys are string (api_id) and values are db.FindKeyAuthsByIdsRow (has both KeyAuthID and ApiID).
	ApiToKeyAuthRow cache.Cache[cache.ScopedKey, db.FindKeyAuthsByIdsRow]

	// ExternalIdToIdentity caches external_id to identity mappings.
	// Keys are string (external_id) scoped to the workspace ID and values are db.Identity (full object).
	ExternalIdToIdentity cache.Cache[cache.ScopedKey, db.Identity]

	// Identity caches identity_id to identity mappings.
	// Keys are string (identity_id) and values are db.Identity.
	Identity cache.Cache[cache.ScopedKey, db.Identity]
}

// Config defines the configuration options for initializing caches.
type Config struct {
	// Logger is used for logging cache operations and errors.
	Logger logging.Logger

	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock

	// Topic for distributed cache invalidation
	CacheInvalidationTopic *eventstream.Topic[*cachev1.CacheInvalidationEvent]

	// NodeID identifies this node in the cluster (defaults to hostname)
	NodeID string
}

// createCache creates a cache instance with optional clustering support.
//
// This is a generic helper function that:
// 1. Creates a local cache with the provided configuration
// 2. If a CacheInvalidationTopic is provided, wraps it with clustering for distributed invalidation
// 3. Returns the cache (either local or clustered)
//
// Type parameters:
//   - K: The key type (must be comparable)
//   - V: The value type to be stored in the cache
//
// Parameters:
//   - config: The main configuration containing clustering settings
//   - cacheConfig: The specific cache configuration (freshness, staleness, size, etc.)
//   - keyToString: Optional converter from key type to string for serialization
//   - stringToKey: Optional converter from string to key type for deserialization
//
// Returns:
//   - cache.Cache[K, V]: The initialized cache instance
//   - error: An error if cache creation failed
func createCache[K comparable, V any](
	config Config,
	cacheConfig cache.Config[K, V],
	keyToString func(K) string,
	stringToKey func(string) (K, error),
) (cache.Cache[K, V], error) {
	// Create local cache
	localCache, err := cache.New(cacheConfig)
	if err != nil {
		return nil, err
	}

	// If no clustering topic is provided, return the local cache
	if config.CacheInvalidationTopic == nil {
		return localCache, nil
	}

	// Wrap with clustering for distributed invalidation
	// The cluster cache will automatically subscribe to invalidation events
	clusterCache, err := clustering.New(clustering.Config[K, V]{
		LocalCache:  localCache,
		Topic:       config.CacheInvalidationTopic,
		Logger:      config.Logger,
		NodeID:      config.NodeID,
		KeyToString: keyToString,
		StringToKey: stringToKey,
	})
	if err != nil {
		return nil, err
	}

	return clusterCache, nil
}

// New creates and initializes all cache instances with appropriate settings.
//
// It configures each cache with specific freshness/staleness windows, size limits,
// resource names for tracing, and wraps them with distributed invalidation if configured.
//
// Parameters:
//   - config: Configuration options including logger, clock, and optional topic for distributed invalidation.
//
// Returns:
//   - Caches: A struct containing all initialized cache instances.
//   - error: An error if any cache failed to initialize.
//
// All caches are thread-safe and can be accessed concurrently. If a CacheInvalidationTopic
// is provided, the caches will automatically handle distributed cache invalidation across
// cluster nodes when entries are modified.
//
// Example:
//
//	logger := logging.NewLogger()
//	clock := clock.RealClock{}
//
//	caches, err := caches.New(caches.Config{
//	    Logger: logger,
//	    Clock: clock,
//	    CacheInvalidationTopic: topic, // optional for distributed invalidation
//	})
//	if err != nil {
//	    log.Fatalf("Failed to initialize caches: %v", err)
//	}
//
//	// Use the caches - invalidation is automatic
//	key, err := caches.KeyByHash.Get(ctx, "some-hash")
func New(config Config) (Caches, error) {
	// Start the global invalidation manager if clustering is enabled
	if config.CacheInvalidationTopic != nil {
		clustering.GetManager().Start(config.CacheInvalidationTopic, config.Logger)
	}

	// Create ratelimit namespace cache (uses ScopedKey)
	ratelimitNamespace, err := createCache(
		config,
		cache.Config[cache.ScopedKey, db.FindRatelimitNamespace]{
			Fresh:    time.Minute,
			Stale:    24 * time.Hour,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "ratelimit_namespace",
			Clock:    config.Clock,
		},
		cache.ScopedKeyToString,
		cache.ScopedKeyFromString,
	)
	if err != nil {
		return Caches{}, err
	}

	// Create verification key cache (uses string keys, no conversion needed)
	verificationKeyByHash, err := createCache(
		config,
		cache.Config[string, db.FindKeyForVerificationRow]{
			Fresh:    10 * time.Second,
			Stale:    10 * time.Minute,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "verification_key_by_hash",
			Clock:    config.Clock,
		},
		nil, // String keys don't need custom converters
		nil,
	)
	if err != nil {
		return Caches{}, err
	}

	// Create API cache (uses ScopedKey)
	liveApiByID, err := createCache(
		config,
		cache.Config[cache.ScopedKey, db.FindLiveApiByIDRow]{
			Fresh:    10 * time.Second,
			Stale:    24 * time.Hour,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "live_api_by_id",
			Clock:    config.Clock,
		},
		cache.ScopedKeyToString,
		cache.ScopedKeyFromString,
	)
	if err != nil {
		return Caches{}, err
	}

	clickhouseSetting, err := createCache(
		config,
		cache.Config[string, db.ClickhouseWorkspaceSetting]{
			Fresh:    10 * time.Second,
			Stale:    24 * time.Hour,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "clickhouse_setting",
			Clock:    config.Clock,
		},
		nil,
		nil,
	)
	if err != nil {
		return Caches{}, err
	}

	// Create key_auth_id -> api row cache
	keyAuthToApiRow, err := createCache(
		config,
		cache.Config[cache.ScopedKey, db.FindKeyAuthsByIdsRow]{
			Fresh:    10 * time.Minute,
			Stale:    24 * time.Hour,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "key_auth_to_api_row",
			Clock:    config.Clock,
		},
		cache.ScopedKeyToString,
		cache.ScopedKeyFromString,
	)
	if err != nil {
		return Caches{}, err
	}

	// Create api_id -> key_auth row cache
	apiToKeyAuthRow, err := createCache(
		config,
		cache.Config[cache.ScopedKey, db.FindKeyAuthsByIdsRow]{
			Fresh:    10 * time.Minute,
			Stale:    24 * time.Hour,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "api_to_key_auth_row",
			Clock:    config.Clock,
		},
		cache.ScopedKeyToString,
		cache.ScopedKeyFromString,
	)
	if err != nil {
		return Caches{}, err
	}

	// Create external_id -> identity mapping cache (scoped to workspace)
	externalIdToIdentity, err := createCache(
		config,
		cache.Config[cache.ScopedKey, db.Identity]{
			Fresh:    10 * time.Minute,
			Stale:    24 * time.Hour,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "external_id_to_identity",
			Clock:    config.Clock,
		},
		cache.ScopedKeyToString,
		cache.ScopedKeyFromString,
	)
	if err != nil {
		return Caches{}, err
	}

	identity, err := createCache(
		config,
		cache.Config[cache.ScopedKey, db.Identity]{
			Fresh:    10 * time.Minute,
			Stale:    24 * time.Hour,
			Logger:   config.Logger,
			MaxSize:  1_000_000,
			Resource: "identity_id_to_external_id",
			Clock:    config.Clock,
		},
		cache.ScopedKeyToString,
		cache.ScopedKeyFromString,
	)
	if err != nil {
		return Caches{}, err
	}

	return Caches{
		RatelimitNamespace:    middleware.WithTracing(ratelimitNamespace),
		LiveApiByID:           middleware.WithTracing(liveApiByID),
		VerificationKeyByHash: middleware.WithTracing(verificationKeyByHash),
		ClickhouseSetting:     middleware.WithTracing(clickhouseSetting),
		KeyAuthToApiRow:       middleware.WithTracing(keyAuthToApiRow),
		ApiToKeyAuthRow:       middleware.WithTracing(apiToKeyAuthRow),
		ExternalIdToIdentity:  middleware.WithTracing(externalIdToIdentity),
		Identity:              middleware.WithTracing(identity),
	}, nil
}
