package router

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ Service = (*service)(nil)

type service struct {
	logger logging.Logger
	db     db.Database
	clock  clock.Clock

	// Cache deployment configs
	deploymentCache cache.Cache[string, *Deployment]
}

// New creates a new router service
func New(cfg Config) (*service, error) {
	deploymentCache, err := cache.New[string, *Deployment](cache.Config[string, *Deployment]{
		Clock:   cfg.Clock,
		MaxSize: 1000,
		Fresh:   10 * time.Second,
		Stale:   60 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &service{
		logger:          cfg.Logger,
		db:              cfg.DB,
		clock:           cfg.Clock,
		deploymentCache: deploymentCache,
	}, nil
}

// GetDeployment returns the deployment config including target address and middlewares
func (s *service) GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error) {
	deployment, hit, err := s.deploymentCache.SWR(ctx, deploymentID, func(ctx context.Context) (*Deployment, error) {
		// TODO: implement actual DB lookup
		// Query deployment table to get:
		// - target k8s service address
		// - middleware configuration
		// - timeout settings
		// - etc.

		// Placeholder - implement when schema is ready
		return nil, fault.New("not implemented",
			fault.Internal("deployment lookup not yet implemented"),
		)
	}, caches.DefaultFindFirstOp)

	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to get deployment"))
	}

	if hit == cache.Null || deployment == nil {
		return nil, fault.New("deployment not found",
			fault.Internal("no deployment found for ID"),
			fault.Public("Service not available"),
		)
	}

	return deployment, nil
}
