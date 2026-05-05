package caches

import (
	"fmt"
	"os"
	"time"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/bus"
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
	RatelimitNamespace cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]

	// VerificationKeyByHash caches verification key lookups by their hash with pre-parsed data.
	VerificationKeyByHash cache.Cache[string, keysdb.CachedKeyData]

	// LiveApiByID caches live API lookups by ID.
	LiveApiByID cache.Cache[cache.ScopedKey, db.FindLiveApiByIDRow]

	// ClickhouseSetting caches clickhouse configuration lookups by workspace ID.
	ClickhouseSetting cache.Cache[string, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow]

	// KeyAuthToApiRow caches key_auth_id to api row mappings.
	KeyAuthToApiRow cache.Cache[cache.ScopedKey, db.FindKeyAuthsByKeyAuthIdsRow]

	// ApiToKeyAuthRow caches api_id to key_auth row mappings.
	ApiToKeyAuthRow cache.Cache[cache.ScopedKey, db.FindKeyAuthsByIdsRow]

	// WorkspaceQuota caches workspace quota lookups by workspace ID.
	WorkspaceQuota cache.Cache[string, keysdb.Quotas]

	// PortalSession caches portal session lookups by session token.
	// Keys are string (session token ID) and values are db.PortalSession.
	// Short fresh window because sessions can expire; stale window allows
	// serving slightly-stale data while revalidating in the background.
	PortalSession cache.Cache[string, db.PortalSession]
}

// Close is a no-op kept for API stability. Cache subscriptions are
// torn down by closing the bus, which is owned by the caller.
func (c *Caches) Close() error { return nil }

// Config defines the configuration options for initializing caches.
type Config struct {
	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock

	// Bus is the event bus used for distributed cache invalidation. Pass
	// bus.NewNoop() in processes that have no gossip configured; the
	// wrapper still works correctly, just without cross-pod fan-out.
	Bus bus.Bus

	// NodeID identifies this node in the cluster.
	NodeID string
}

type clusterOpts[K comparable] struct {
	bus         bus.Bus
	nodeID      string
	keyToString func(K) string
	stringToKey func(string) (K, error)
}

// createCache wraps a local cache with cluster invalidation. opts.bus must
// be non-nil; pass bus.NewNoop() to disable cross-pod fan-out without
// changing the call shape.
func createCache[K comparable, V any](
	cacheConfig cache.Config[K, V],
	opts clusterOpts[K],
) (cache.Cache[K, V], error) {
	localCache, err := cache.New(cacheConfig)
	if err != nil {
		return nil, err
	}

	clusterCache, err := clustering.New(clustering.Config[K, V]{
		LocalCache:  localCache,
		Bus:         opts.bus,
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
func New(config Config) (Caches, error) {
	if config.Bus == nil {
		config.Bus = bus.NewNoop()
	}

	if config.NodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		config.NodeID = fmt.Sprintf("%s-%s", hostname, uid.New("node"))
	}

	scopedKeyOpts := clusterOpts[cache.ScopedKey]{
		bus:         config.Bus,
		nodeID:      config.NodeID,
		keyToString: cache.ScopedKeyToString,
		stringToKey: cache.ScopedKeyFromString,
	}
	stringKeyOpts := clusterOpts[string]{
		bus:         config.Bus,
		nodeID:      config.NodeID,
		keyToString: nil, // defaults handle string keys
		stringToKey: nil,
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
		cache.Config[string, keysdb.CachedKeyData]{
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

	workspaceQuota, err := createCache(
		cache.Config[string, keysdb.Quotas]{
			Fresh:    time.Minute,
			Stale:    24 * time.Hour,
			MaxSize:  100_000,
			Resource: "workspace_quota",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return Caches{}, err
	}

	portalSession, err := createCache(
		cache.Config[string, db.PortalSession]{
			Fresh:    10 * time.Second,
			Stale:    5 * time.Minute,
			MaxSize:  100_000,
			Resource: "portal_session",
			Clock:    config.Clock,
		},
		stringKeyOpts,
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
		WorkspaceQuota:        middleware.WithTracing(workspaceQuota),
		PortalSession:         middleware.WithTracing(portalSession),
	}, nil
}
