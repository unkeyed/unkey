package caches

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Caches holds all cache instances used throughout the application.
// Each field represents a specialized cache for a specific data entity.
type Caches struct {
	// RatelimitNamespaceByName caches ratelimit namespace lookups by name.
	// Keys are db.FindRatelimitNamespaceByNameParams and values are db.RatelimitNamespace.
	RatelimitNamespaceByName cache.Cache[db.FindRatelimitNamespaceByNameParams, db.RatelimitNamespace]

	// RatelimitOverridesMatch caches ratelimit override matches for specific criteria.
	// Keys are db.ListRatelimitOverrideMatchesParams and values are slices of db.RatelimitOverride.
	RatelimitOverridesMatch cache.Cache[db.ListRatelimitOverrideMatchesParams, []db.RatelimitOverride]

	// KeyByHash caches API key lookups by their hash.
	// Keys are string (hash) and values are db.Key.
	KeyByHash cache.Cache[string, db.Key]

	// PermissionsByKeyId caches permission strings for a given key ID.
	// Keys are string (key ID) and values are slices of string representing permissions.
	PermissionsByKeyId cache.Cache[string, []string]

	// WorkspaceByID caches workspace lookups by their ID.
	// Keys are string (workspace ID) and values are db.Workspace.
	WorkspaceByID cache.Cache[string, db.Workspace]

	ApiByID cache.Cache[string, db.Api]

	IdentityByID cache.Cache[string, db.Identity]
}

// Config defines the configuration options for initializing caches.
type Config struct {
	// Logger is used for logging cache operations and errors.
	Logger logging.Logger

	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock
}

// New creates and initializes all cache instances with appropriate settings.
//
// It configures each cache with specific freshness/staleness windows, size limits,
// resource names for tracing, and wraps them with tracing middleware.
//
// Parameters:
//   - config: Configuration options including logger and clock implementations.
//
// Returns:
//   - Caches: A struct containing all initialized cache instances.
//   - error: An error if any cache failed to initialize.
//
// All caches are thread-safe and can be accessed concurrently.
//
// Example:
//
//	logger := logging.NewLogger()
//	clock := clock.RealClock{}
//
//	caches, err := caches.New(caches.Config{
//	    Logger: logger,
//	    Clock: clock,
//	})
//	if err != nil {
//	    log.Fatalf("Failed to initialize caches: %v", err)
//	}
//
//	// Use the caches
//	key, err := caches.KeyByHash.Get(ctx, "some-hash")
func New(config Config) (Caches, error) {

	ratelimitNamespace, err := cache.New(cache.Config[db.FindRatelimitNamespaceByNameParams, db.RatelimitNamespace]{
		Fresh:    time.Minute,
		Stale:    24 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "ratelimit_namespace_by_name",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	ratelimitOverridesMatch, err := cache.New(cache.Config[db.ListRatelimitOverrideMatchesParams, []db.RatelimitOverride]{
		Fresh:    time.Minute,
		Stale:    24 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "ratelimit_overrides",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	keyByHash, err := cache.New(cache.Config[string, db.Key]{
		Fresh:   10 * time.Second,
		Stale:   24 * time.Hour,
		Logger:  config.Logger,
		MaxSize: 1_000_000,

		Resource: "key_by_hash",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	permissionsByKeyId, err := cache.New(cache.Config[string, []string]{
		Fresh:   10 * time.Second,
		Stale:   24 * time.Hour,
		Logger:  config.Logger,
		MaxSize: 1_000_000,

		Resource: "permissions_by_key_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	workspaceByID, err := cache.New(cache.Config[string, db.Workspace]{
		Fresh:   10 * time.Second,
		Stale:   24 * time.Hour,
		Logger:  config.Logger,
		MaxSize: 1_000_000,

		Resource: "workspace_by_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	apiById, err := cache.New(cache.Config[string, db.Api]{
		Fresh:   10 * time.Second,
		Stale:   24 * time.Hour,
		Logger:  config.Logger,
		MaxSize: 1_000_000,

		Resource: "api_by_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	identityByID, err := cache.New(cache.Config[string, db.Identity]{
		Fresh:   10 * time.Second,
		Stale:   24 * time.Hour,
		Logger:  config.Logger,
		MaxSize: 1_000_000,

		Resource: "identity_by_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	return Caches{
		RatelimitNamespaceByName: middleware.WithTracing(ratelimitNamespace),
		RatelimitOverridesMatch:  middleware.WithTracing(ratelimitOverridesMatch),
		KeyByHash:                middleware.WithTracing(keyByHash),
		PermissionsByKeyId:       middleware.WithTracing(permissionsByKeyId),
		WorkspaceByID:            middleware.WithTracing(workspaceByID),
		ApiByID:                  middleware.WithTracing(apiById),
		IdentityByID:             middleware.WithTracing(identityByID),
	}, nil
}
