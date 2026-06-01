package caches

import (
	"fmt"
	"os"
	"time"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
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
	// Keys are string (hash) and values are keysdb.CachedKeyData (includes pre-parsed IP whitelist).
	VerificationKeyByHash cache.Cache[string, keysdb.CachedKeyData]

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

	// WorkspaceQuota caches workspace quota lookups by workspace ID.
	// Keys are string (workspace ID) and values are keysdb.Quotas.
	WorkspaceQuota cache.Cache[string, keysdb.Quotas]

	// PortalSession caches portal session lookups by session token.
	// Keys are string (session token ID) and values are db.PortalSession.
	// Short fresh window because sessions can expire; stale window allows
	// serving slightly-stale data while revalidating in the background.
	PortalSession cache.Cache[string, db.PortalSession]
}

// Close shuts down the caches and cleans up resources.
func (c *Caches) Close() error {
	return nil
}

// Config defines the configuration options for initializing caches.
type Config struct {
	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock

	// NodeID identifies this node (defaults to hostname-uniqueid to ensure uniqueness).
	NodeID string
}

// New creates and initializes all cache instances with appropriate settings.
//
// It configures each cache with specific freshness/staleness windows, size limits,
// and resource names for tracing.
func New(config Config) (Caches, error) {
	if config.NodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		config.NodeID = fmt.Sprintf("%s-%s", hostname, uid.New("node"))
	}

	ratelimitNamespace, err := cache.New(cache.Config[cache.ScopedKey, db.FindRatelimitNamespace]{
		Fresh:    time.Minute,
		Stale:    24 * time.Hour,
		MaxSize:  1_000_000,
		Resource: "ratelimit_namespace",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	verificationKeyByHash, err := cache.New(cache.Config[string, keysdb.CachedKeyData]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		MaxSize:  1_000_000,
		Resource: "verification_key_by_hash",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	liveApiByID, err := cache.New(cache.Config[cache.ScopedKey, db.FindLiveApiByIDRow]{
		Fresh:    10 * time.Second,
		Stale:    24 * time.Hour,
		MaxSize:  1_000_000,
		Resource: "live_api_by_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	clickhouseSetting, err := cache.New(cache.Config[string, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow]{
		Fresh:    time.Minute,
		Stale:    24 * time.Hour,
		MaxSize:  1_000_000,
		Resource: "clickhouse_setting",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	keyAuthToApiRow, err := cache.New(cache.Config[cache.ScopedKey, db.FindKeyAuthsByKeyAuthIdsRow]{
		Fresh:    10 * time.Minute,
		Stale:    24 * time.Hour,
		MaxSize:  1_000_000,
		Resource: "key_auth_to_api_row",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	apiToKeyAuthRow, err := cache.New(cache.Config[cache.ScopedKey, db.FindKeyAuthsByIdsRow]{
		Fresh:    10 * time.Minute,
		Stale:    24 * time.Hour,
		MaxSize:  1_000_000,
		Resource: "api_to_key_auth_row",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	workspaceQuota, err := cache.New(cache.Config[string, keysdb.Quotas]{
		Fresh:    time.Minute,
		Stale:    24 * time.Hour,
		MaxSize:  100_000,
		Resource: "workspace_quota",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	portalSession, err := cache.New(cache.Config[string, db.PortalSession]{
		Fresh:    10 * time.Second,
		Stale:    5 * time.Minute,
		MaxSize:  100_000,
		Resource: "portal_session",
		Clock:    config.Clock,
	})
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
