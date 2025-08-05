package routing

import (
	"context"
	"fmt"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
)

// routingService implements the RoutingService interface with database backend.
type routingService struct {
	db     db.Database
	logger logging.Logger
	cache  cache.Cache[string, *TargetInfo]
	// Simple counter for round-robin VM selection
	vmCounter uint64
}

var _ Service = (*routingService)(nil)

// Config holds configuration for the routing service.
type Config struct {
	DB     db.Database
	Logger logging.Logger
	Clock  clock.Clock
}

// New creates a new routing service instance.
func New(config Config) (Service, error) {
	// Create cache for target configurations
	targetCache, err := cache.New(cache.Config[string, *TargetInfo]{
		Fresh:    5 * time.Minute,  // Data is fresh for 5 minutes
		Stale:    30 * time.Minute, // Serve stale data for up to 30 minutes while refreshing
		Logger:   config.Logger,
		MaxSize:  10_000, // Cache up to 10k target configurations
		Resource: "routing_targets",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create routing cache: %w", err)
	}

	return &routingService{
		db:     config.DB,
		logger: config.Logger,
		cache:  middleware.WithTracing(targetCache),
	}, nil
}

// GetTarget retrieves target configuration by ID.
func (s *routingService) GetTarget(ctx context.Context, targetID string) (*TargetInfo, error) {
	// Use SWR (Stale-While-Revalidate) cache pattern
	targetInfo, _, err := s.cache.SWR(ctx, targetID, func(ctx context.Context) (*TargetInfo, error) {
		s.logger.Debug("fetching target from database", "targetID", targetID)

		// For now, return a mock gateway since we don't have the actual DB schema
		// In production, this would query the database for gateway configuration
		target := &TargetInfo{
			GatewayID:    targetID,
			WorkspaceID:  "default-workspace",
			Name:         fmt.Sprintf("Gateway %s", targetID),
			Enabled:      true,
			AvailableVMs: []string{"http://httpbin.org", "http://httpbin.org"}, // Mock VMs
			Metadata: map[string]string{
				"version": "1.0",
			},
		}

		return target, nil
	}, func(err error) cache.Op {
		if err != nil {
			if db.IsNotFound(err) {
				return cache.WriteNull
			}

			return cache.Noop
		}

		return cache.WriteValue
	})

	return targetInfo, err
}

// GetTargetByHost finds target configuration based on the request host.
func (s *routingService) GetTargetByHost(ctx context.Context, host string) (*TargetInfo, error) {
	s.logger.Debug("resolving target by host", "host", host)

	// Use cache key based on host for efficient lookups
	cacheKey := fmt.Sprintf("host:%s", host)

	// Use SWR cache pattern to get target for this host
	targetInfo, _, err := s.cache.SWR(ctx, cacheKey, func(ctx context.Context) (*TargetInfo, error) {
		// For now, return a default target
		// In production, this would query the database to find which target handles this host
		return s.GetTarget(ctx, "default-target")
	}, func(err error) cache.Op {
		if err != nil {
			if db.IsNotFound(err) {
				return cache.WriteNull
			}

			return cache.Noop
		}

		return cache.WriteValue
	})

	return targetInfo, err
}

// SelectVM picks an available VM from the gateway's VM list using simple round-robin.
func (s *routingService) SelectVM(ctx context.Context, targetInfo *TargetInfo) (*url.URL, error) {
	s.logger.Debug("selecting VM for gateway",
		"gatewayID", targetInfo.GatewayID,
		"availableVMs", len(targetInfo.AvailableVMs),
	)

	if !targetInfo.Enabled {
		return nil, fmt.Errorf("gateway %s is disabled", targetInfo.GatewayID)
	}

	if len(targetInfo.AvailableVMs) == 0 {
		return nil, fmt.Errorf("no VMs available for gateway %s", targetInfo.GatewayID)
	}

	// Simple round-robin selection
	index := atomic.AddUint64(&s.vmCounter, 1) - 1
	selectedVM := targetInfo.AvailableVMs[index%uint64(len(targetInfo.AvailableVMs))]

	targetURL, err := url.Parse(selectedVM)
	if err != nil {
		return nil, fmt.Errorf("invalid VM URL %s: %w", selectedVM, err)
	}

	s.logger.Debug("selected VM",
		"gatewayID", targetInfo.GatewayID,
		"selectedVM", targetURL.String(),
	)

	return targetURL, nil
}
