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
	// RatelimitNamespace caches ratelimit namespace lookups by name or ID.
	// Keys are cache.ScopedKey and values are db.FindRatelimitNamespace.
	RatelimitNamespace cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]

	// VerificationKeyByHash caches verification key lookups by their hash with pre-parsed data.
	// Keys are string (hash) and values are db.CachedKeyData (includes pre-parsed IP whitelist).
	VerificationKeyByHash cache.Cache[string, db.CachedKeyData]

	// LiveApiByID caches live API lookups by ID.
	// Keys are string (ID) and values are db.FindLiveApiByIDRow.
	LiveApiByID cache.Cache[string, db.FindLiveApiByIDRow]

	// LiveKeyByID caches live key lookups by ID.
	// Keys are string (ID) and values are db.FindLiveKeyByIDRow.
	LiveKeyByID cache.Cache[string, db.FindLiveKeyByIDRow]
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
	ratelimitNamespace, err := cache.New(cache.Config[cache.ScopedKey, db.FindRatelimitNamespace]{
		Fresh:    time.Minute,
		Stale:    24 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "ratelimit_namespace",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	verificationKeyByHash, err := cache.New(cache.Config[string, db.CachedKeyData]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "verification_key_by_hash",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	liveApiByID, err := cache.New(cache.Config[string, db.FindLiveApiByIDRow]{
		Fresh:    10 * time.Second,
		Stale:    24 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "live_api_by_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	liveKeyByID, err := cache.New(cache.Config[string, db.FindLiveKeyByIDRow]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "live_key_by_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	return Caches{
		RatelimitNamespace:    middleware.WithTracing(ratelimitNamespace),
		LiveApiByID:           middleware.WithTracing(liveApiByID),
		VerificationKeyByHash: middleware.WithTracing(verificationKeyByHash),
		LiveKeyByID:           middleware.WithTracing(liveKeyByID),
	}, nil
}
