package router

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ Service = (*service)(nil)

type service struct {
	logger        logging.Logger
	db            db.Database
	clock         clock.Clock
	environmentID string
	region        string

	deploymentCache cache.Cache[string, db.Deployment]
	instancesCache  cache.Cache[string, []db.Instance]
}

func New(cfg Config) (*service, error) {
	deploymentCache, err := cache.New[string, db.Deployment](cache.Config[string, db.Deployment]{
		Clock:   cfg.Clock,
		MaxSize: 1000,
		Fresh:   10 * time.Second,
		Stale:   60 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	instancesCache, err := cache.New[string, []db.Instance](cache.Config[string, []db.Instance]{
		Clock:   cfg.Clock,
		MaxSize: 1000,
		Fresh:   5 * time.Second,
		Stale:   30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &service{
		logger:          cfg.Logger,
		db:              cfg.DB,
		clock:           cfg.Clock,
		environmentID:   cfg.EnvironmentID,
		region:          cfg.Region,
		deploymentCache: deploymentCache,
		instancesCache:  instancesCache,
	}, nil
}

func (s *service) GetDeployment(ctx context.Context, deploymentID string) (db.Deployment, error) {
	deployment, hit, err := s.deploymentCache.SWR(ctx, deploymentID, func(ctx context.Context) (db.Deployment, error) {
		return db.Query.FindDeploymentByID(ctx, s.db.RO(), deploymentID)
	}, caches.DefaultFindFirstOp)
	if err != nil && !db.IsNotFound(err) {
		return db.Deployment{}, fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.InternalServerError.URN()),
			fault.Internal("failed to get deployment"),
		)
	}

	if hit == cache.Null || db.IsNotFound(err) {
		return db.Deployment{}, fault.New("deployment not found",
			fault.Code(codes.Gateway.Routing.DeploymentNotFound.URN()),
			fault.Internal("no deployment found for ID or wrong environment"),
			fault.Public("Deployment not found"),
		)
	}

	if deployment.EnvironmentID != s.environmentID {
		s.logger.Warn("deployment does not belong to this environment",
			"deploymentID", deploymentID,
			"deploymentEnv", deployment.EnvironmentID,
			"gatewayEnv", s.environmentID,
		)

		// Return as not found to avoid leaking information about deployments in other environments
		return db.Deployment{}, fault.New("deployment not found",
			fault.Code(codes.Gateway.Routing.DeploymentNotFound.URN()),
			fault.Internal(fmt.Sprintf("deployment %s belongs to environment %s, but gateway serves %s", deploymentID, deployment.EnvironmentID, s.environmentID)),
			fault.Public("Deployment not found"),
		)
	}

	return deployment, nil
}

func (s *service) SelectInstance(ctx context.Context, deploymentID string) (db.Instance, error) {
	instances, hit, err := s.instancesCache.SWR(ctx, deploymentID, func(ctx context.Context) ([]db.Instance, error) {
		return db.Query.FindInstancesByDeploymentIdAndRegion(
			ctx,
			s.db.RO(),
			db.FindInstancesByDeploymentIdAndRegionParams{
				Deploymentid: deploymentID,
				Region:       s.region,
			},
		)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		return db.Instance{}, fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.InternalServerError.URN()),
			fault.Internal("failed to get instances"),
		)
	}

	if hit == cache.Null || len(instances) == 0 {
		return db.Instance{}, fault.New("no instances found",
			fault.Code(codes.Gateway.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no instances for deployment %s in region %s", deploymentID, s.region)),
			fault.Public("Service temporarily unavailable"),
		)
	}

	var runningInstances []db.Instance
	for _, instance := range instances {
		if instance.Status == db.InstancesStatusRunning {
			runningInstances = append(runningInstances, instance)
		}
	}

	if len(runningInstances) == 0 {
		return db.Instance{}, fault.New("no running instances",
			fault.Code(codes.Gateway.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no running instances for deployment %s in region %s (found %d total)", deploymentID, s.region, len(instances))),
			fault.Public("Service temporarily unavailable"),
		)
	}

	randomIndex := rand.IntN(len(runningInstances))
	return runningInstances[randomIndex], nil
}
