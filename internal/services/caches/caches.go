package caches

import (
	"fmt"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
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
	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock

	// Broadcaster for distributed cache invalidation via gossip.
	// If nil, caches operate in local-only mode (no distributed invalidation).
	Broadcaster clustering.Broadcaster

	// NodeID identifies this node in the cluster (defaults to hostname-uniqueid to ensure uniqueness)
	NodeID string
}

// clusterOpts bundles the dispatcher and key converter functions needed for
// distributed cache invalidation. These are coupled because converters are only
// meaningful when clustering is enabled (i.e., when a dispatcher exists).
// Pass nil when clustering is disabled.
type clusterOpts[K comparable] struct {
	dispatcher  *clustering.InvalidationDispatcher
	broadcaster clustering.Broadcaster
	nodeID      string
	keyToString func(K) string
	stringToKey func(string) (K, error)
}

// createCache creates a cache instance with optional clustering support.
//
// This is a generic helper function that:
// 1. Creates a local cache with the provided configuration
// 2. If clustering opts are provided, wraps it with clustering for distributed invalidation
// 3. Returns the cache (either local or clustered)
func createCache[K comparable, V any](
	cacheConfig cache.Config[K, V],
	opts *clusterOpts[K],
) (cache.Cache[K, V], error) {
	// Create local cache
	localCache, err := cache.New(cacheConfig)
	if err != nil {
		return nil, err
	}

	// If no clustering is enabled, return the local cache directly.
	// This avoids the ClusterCache wrapper overhead when clustering isn't needed,
	// keeping cache operations (Get/Set/etc) as fast as possible on the hot path.
	if opts == nil {
		return localCache, nil
	}

	// Wrap with clustering for distributed invalidation
	// The cluster cache will automatically register with the dispatcher
	clusterCache, err := clustering.New(clustering.Config[K, V]{
		LocalCache:  localCache,
		Broadcaster: opts.broadcaster,
		Dispatcher:  opts.dispatcher,
		NodeID:      opts.nodeID,
		KeyToString: opts.keyToString,
		StringToKey: opts.stringToKey,
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

	// Build clustering options if a broadcaster is configured.
	// When nil, createCache returns unwrapped local caches (no clustering overhead).
	var dispatcher *clustering.InvalidationDispatcher
	var scopedKeyOpts *clusterOpts[cache.ScopedKey]
	var stringKeyOpts *clusterOpts[string]

	if config.Broadcaster != nil {
		var err error
		dispatcher, err = clustering.NewInvalidationDispatcher(config.Broadcaster)
		if err != nil {
			return Caches{}, err
		}

		scopedKeyOpts = &clusterOpts[cache.ScopedKey]{
			dispatcher:  dispatcher,
			broadcaster: config.Broadcaster,
			nodeID:      config.NodeID,
			keyToString: cache.ScopedKeyToString,
			stringToKey: cache.ScopedKeyFromString,
		}
		stringKeyOpts = &clusterOpts[string]{
			dispatcher:  dispatcher,
			broadcaster: config.Broadcaster,
			nodeID:      config.NodeID,
			keyToString: nil, // defaults handle string keys
			stringToKey: nil,
		}
	}

	// Ensure the dispatcher is closed if any subsequent cache creation fails.
	initialized := false
	if dispatcher != nil {
		defer func() {
			if !initialized {
				_ = dispatcher.Close()
			}
		}()
	}

	ratelimitNamespace, err := createCache(
		cache.Config[cache.ScopedKey, db.FindRatelimitNamespace]{
			Fresh:    time.Minute,
			Stale:    24 * time.Hour,
			MaxSize:  1_000_000,
			Resource: "ratelimit_namespace",
			Clock:    config.Clock,
		},
		scopedKeyOpts,
	)
	if err != nil {
		return Caches{}, err
	}

	verificationKeyByHash, err := createCache(
		cache.Config[string, db.CachedKeyData]{
			Fresh:    10 * time.Second,
			Stale:    10 * time.Minute,
			MaxSize:  1_000_000,
			Resource: "verification_key_by_hash",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return Caches{}, err
	}

	liveApiByID, err := createCache(
		cache.Config[cache.ScopedKey, db.FindLiveApiByIDRow]{
			Fresh:    10 * time.Second,
			Stale:    24 * time.Hour,
			MaxSize:  1_000_000,
			Resource: "live_api_by_id",
			Clock:    config.Clock,
		},
		scopedKeyOpts,
	)
	if err != nil {
		return Caches{}, err
	}

	clickhouseSetting, err := createCache(
		cache.Config[string, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow]{
			Fresh:    time.Minute,
			Stale:    24 * time.Hour,
			MaxSize:  1_000_000,
			Resource: "clickhouse_setting",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return Caches{}, err
	}

	keyAuthToApiRow, err := createCache(
		cache.Config[cache.ScopedKey, db.FindKeyAuthsByKeyAuthIdsRow]{
			Fresh:    10 * time.Minute,
			Stale:    24 * time.Hour,
			MaxSize:  1_000_000,
			Resource: "key_auth_to_api_row",
			Clock:    config.Clock,
		},
		scopedKeyOpts,
	)
	if err != nil {
		return Caches{}, err
	}

	apiToKeyAuthRow, err := createCache(
		cache.Config[cache.ScopedKey, db.FindKeyAuthsByIdsRow]{
			Fresh:    10 * time.Minute,
			Stale:    24 * time.Hour,
			MaxSize:  1_000_000,
			Resource: "api_to_key_auth_row",
			Clock:    config.Clock,
		},
		scopedKeyOpts,
	)
	if err != nil {
		return Caches{}, err
	}

	initialized = true
	return Caches{
		RatelimitNamespace:    middleware.WithTracing(ratelimitNamespace),
		LiveApiByID:           middleware.WithTracing(liveApiByID),
		VerificationKeyByHash: middleware.WithTracing(verificationKeyByHash),
		ClickhouseSetting:     middleware.WithTracing(clickhouseSetting),
		KeyAuthToApiRow:       middleware.WithTracing(keyAuthToApiRow),
		ApiToKeyAuthRow:       middleware.WithTracing(apiToKeyAuthRow),
		dispatcher:            dispatcher,
	}, nil
}
