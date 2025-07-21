package caches

import (
	"context"
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
	// RatelimitNamespaceByName caches ratelimit namespace lookups by name.
	// Keys are string and values are db.FindRatelimitNamespace.
	RatelimitNamespaceByName cache.Cache[string, db.FindRatelimitNamespace]

	// VerificationKeyByHash caches verification key lookups by their hash.
	// Keys are string (hash) and values are db.VerificationKey.
	VerificationKeyByHash cache.Cache[string, db.FindKeyForVerificationRow]

	// ApiByID caches API lookups by their ID.
	// Keys are string (ID) and values are db.Api.
	ApiByID cache.Cache[string, db.Api]
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

	// Create local caches first
	ratelimitNamespace, err := cache.New(cache.Config[string, db.FindRatelimitNamespace]{
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

	verificationKeyByHash, err := cache.New(cache.Config[string, db.FindKeyForVerificationRow]{
		Fresh:    30 * time.Second,
		Stale:    24 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "verification_key_by_hash",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	apiById, err := cache.New(cache.Config[string, db.Api]{
		Fresh:    10 * time.Second,
		Stale:    24 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "api_by_id",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	if config.CacheInvalidationTopic != nil {
		// Create cluster caches that automatically handle distributed invalidation
		ratelimitNamespace, err = clustering.New(clustering.Config[db.FindRatelimitNamespace]{
			LocalCache: ratelimitNamespace,
			Topic:      config.CacheInvalidationTopic,
			Logger:     config.Logger,
			NodeID:     config.NodeID,
		})
		if err != nil {
			return Caches{}, err
		}

		verificationKeyByHash, err = clustering.New(clustering.Config[db.FindKeyForVerificationRow]{
			LocalCache: verificationKeyByHash,
			Topic:      config.CacheInvalidationTopic,
			Logger:     config.Logger,
			NodeID:     config.NodeID,
		})
		if err != nil {
			return Caches{}, err
		}

		apiById, err = clustering.New(clustering.Config[db.Api]{
			LocalCache: apiById,
			Topic:      config.CacheInvalidationTopic,
			Logger:     config.Logger,
			NodeID:     config.NodeID,
		})
		if err != nil {
			return Caches{}, err
		}

		// Start consuming invalidation events
		consumer := config.CacheInvalidationTopic.NewConsumer()
		consumer.Consume(context.Background(), func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
			// Don't process our own events to avoid loops
			if event.SourceInstance == config.NodeID {
				return nil
			}

			// Route invalidation events to the appropriate cache
			switch event.CacheName {
			case ratelimitNamespace.Name():
				ratelimitNamespace.Remove(ctx, event.CacheKey)
			case verificationKeyByHash.Name():
				verificationKeyByHash.Remove(ctx, event.CacheKey)
			case apiById.Name():
				apiById.Remove(ctx, event.CacheKey)
			}
			return nil
		})
	}

	return Caches{
		RatelimitNamespaceByName: middleware.WithTracing(ratelimitNamespace),
		ApiByID:                  middleware.WithTracing(apiById),
		VerificationKeyByHash:    middleware.WithTracing(verificationKeyByHash),
	}, nil
}
