package caches

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Caches holds all shared cache instances for the ctrl application.
// Caches holds all shared cache instances for ctrl application.
//
// This struct provides centralized access to performance-critical caches
// used throughout the control plane for ACME operations and
// domain validation.
type Caches struct {
	// Domains cache stores custom domain data for ACME challenges.
	// Reduces database queries during domain validation and ownership checks.
	Domains cache.Cache[string, db.CustomDomain]

	// Challenges cache stores ACME challenge tokens and authorizations.
	// Has short TTL due to rapid state changes during certificate issuance.
	Challenges cache.Cache[string, db.AcmeChallenge]
}

// Config holds configuration for cache initialization.
//
// Provides logger and clock dependencies for cache instances
// with proper configuration for different data types.
type Config struct {
	// Logger for cache operations and error reporting.
	Logger logging.Logger

	// Clock provides time operations for TTL calculations.
	// Uses clock.New() if not provided for production use.
	Clock clock.Clock
}

// New creates configured cache instances for control plane operations.
//
// This function initializes both domain and challenge caches with
// optimized TTL values based on data access patterns. Domains
// use longer TTL due to infrequent changes, while challenges
// use short TTL due to rapid state changes during ACME flows.
//
// Returns configured Caches struct or error if cache creation fails.
func New(cfg Config) (*Caches, error) {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}

	domains, err := cache.New(cache.Config[string, db.CustomDomain]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10000,
		Logger:   cfg.Logger,
		Resource: "domains",
		Clock:    clk,
	})
	if err != nil {
		return nil, err
	}

	// Short TTL for challenges since they change during ACME flow
	challenges, err := cache.New(cache.Config[string, db.AcmeChallenge]{
		Fresh:    10 * time.Second,
		Stale:    30 * time.Second,
		MaxSize:  1000,
		Logger:   cfg.Logger,
		Resource: "acme_challenges",
		Clock:    clk,
	})
	if err != nil {
		return nil, err
	}

	return &Caches{
		Domains:    domains,
		Challenges: challenges,
	}, nil
}
