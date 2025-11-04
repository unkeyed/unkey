package caches

import (
	"fmt"
	"os"
	"time"

	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/clustering"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Caches holds all cache instances used throughout the application.
// Each field represents a specialized cache for a specific data entity.
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

	// dispatcher handles routing of invalidation events to all caches in this process.
	// This is not exported as it's an internal implementation detail.
	dispatcher *clustering.InvalidationDispatcher
}

// Close shuts down the caches and cleans up resources.
func (c *Caches) Close() error {
	// Close the dispatcher to stop consuming invalidation events
	if c.dispatcher != nil {
		return c.dispatcher.Close()
	}

	return nil
}

// Config defines the configuration options for initializing caches.
type Config struct {
	// Logger is used for logging cache operations and errors.
	Logger logging.Logger

	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock

	// Topic for distributed cache invalidation
	CacheInvalidationTopic *eventstream.Topic[*cachev1.CacheInvalidationEvent]

	// NodeID identifies this node in the cluster (defaults to hostname-uniqueid to ensure uniqueness)
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
	dispatcher *clustering.InvalidationDispatcher,
	cacheConfig cache.Config[K, V],
	keyToString func(K) string,
	stringToKey func(string) (K, error),
) (cache.Cache[K, V], error) {
	// Create local cache
	localCache, err := cache.New(cacheConfig)
	if err != nil {
		return nil, err
	}

	// If no clustering is enabled, return the local cache directly.
	// This avoids the ClusterCache wrapper overhead when clustering isn't needed,
	// keeping cache operations (Get/Set/etc) as fast as possible on the hot path.
	if dispatcher == nil {
		return localCache, nil
	}

	// Wrap with clustering for distributed invalidation
	// The cluster cache will automatically register with the dispatcher
	clusterCache, err := clustering.New(clustering.Config[K, V]{
		LocalCache:  localCache,
		Topic:       config.CacheInvalidationTopic,
		Dispatcher:  dispatcher,
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
	// Apply default NodeID if not provided
	// Format: hostname-uniqueid to ensure uniqueness across nodes
	if config.NodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		// Add unique ID to prevent collisions when multiple nodes have same hostname
		config.NodeID = fmt.Sprintf("%s-%s", hostname, uid.New("node"))
	}

	// Create invalidation dispatcher if clustering is enabled.
	// We intentionally leave dispatcher as nil when clustering is disabled to avoid
	// wrapping caches with ClusterCache. This eliminates wrapper overhead on the hot path
	// (cache Get/Set operations) when clustering isn't needed.
	var dispatcher *clustering.InvalidationDispatcher
	if config.CacheInvalidationTopic != nil {
		var err error
		dispatcher, err = clustering.NewInvalidationDispatcher(config.CacheInvalidationTopic, config.Logger)
		if err != nil {
			return Caches{}, err
		}
	}

	// Create ratelimit namespace cache (uses ScopedKey)
	ratelimitNamespace, err := createCache(
		config,
		dispatcher,
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
		dispatcher,
		cache.Config[string, db.CachedKeyData]{
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
		dispatcher,
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

	return Caches{
		RatelimitNamespace:    middleware.WithTracing(ratelimitNamespace),
		LiveApiByID:           middleware.WithTracing(liveApiByID),
		VerificationKeyByHash: middleware.WithTracing(verificationKeyByHash),
		dispatcher:            dispatcher,
	}, nil
}
