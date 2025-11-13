package caches

import (
	"crypto/tls"
	"fmt"
	"time"

	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
)

// Caches holds all cache instances used throughout the application.
// Each field represents a specialized cache for a specific data entity.
type Caches struct {
	// HostName -> Config
	GatewayConfig cache.Cache[string, routing.ConfigWithWorkspace]

	// DeploymentID -> OpenAPI Spec Validator
	OpenAPISpec cache.Cache[string, validator.Validator]

	// VmID -> VM Info
	Instance cache.Cache[string, partitiondb.Instance]

	// HostName -> Certificate
	TLSCertificate cache.Cache[string, tls.Certificate]

	// KeyHash -> Key verification data with pre-parsed IP whitelist (for keys service)
	VerificationKeyByHash cache.Cache[string, db.CachedKeyData]
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
	gatewayConfig, err := cache.New(cache.Config[string, routing.ConfigWithWorkspace]{
		Fresh:    time.Second * 5,
		Stale:    time.Second * 30,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "gateway_config",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create routing cache: %w", err)
	}

	instanceCache, err := cache.New(cache.Config[string, partitiondb.Instance]{
		Fresh:    time.Second * 10,
		Stale:    time.Minute,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "instance",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create instance cache: %w", err)
	}

	tlsCertificate, err := cache.New(cache.Config[string, tls.Certificate]{
		Fresh:    time.Hour,
		Stale:    time.Hour * 12,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "tls_certificate",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create certificate cache: %w", err)
	}

	verificationKeyByHash, err := cache.New(cache.Config[string, db.CachedKeyData]{
		Fresh:    time.Minute,
		Stale:    time.Minute * 10,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "verification_key_by_hash",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, err
	}

	openapiSpec, err := cache.New(cache.Config[string, validator.Validator]{
		Fresh:    30 * time.Minute,
		Stale:    2 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000,
		Resource: "openapi_spec_validator",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create OpenAPI spec cache: %w", err)
	}

	return Caches{
		GatewayConfig:         middleware.WithTracing(gatewayConfig),
		Instance:              middleware.WithTracing(instanceCache),
		TLSCertificate:        middleware.WithTracing(tlsCertificate),
		VerificationKeyByHash: middleware.WithTracing(verificationKeyByHash),
		OpenAPISpec:           middleware.WithTracing(openapiSpec),
	}, nil
}
