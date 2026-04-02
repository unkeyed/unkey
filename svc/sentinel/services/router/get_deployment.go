package router

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

func (s *service) GetDeployment(ctx context.Context, deploymentID string) (db.Deployment, error) {
	t := time.Now()

	deployment, hit, err := s.deploymentCache.SWR(ctx, deploymentID, func(ctx context.Context) (db.Deployment, error) {
		return db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	}, caches.DefaultFindFirstOp)

	sentinelRoutingDuration.WithLabelValues("get_deployment").Observe(time.Since(t).Seconds())

	if err != nil && !db.IsNotFound(err) {
		sentinelDeploymentLookupTotal.WithLabelValues("error").Inc()
		return db.Deployment{}, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("failed to get deployment"),
		)
	}

	if hit == cache.Null || db.IsNotFound(err) {
		sentinelDeploymentLookupTotal.WithLabelValues("not_found").Inc()
		return db.Deployment{}, fault.New("deployment not found",
			fault.Code(codes.Sentinel.Routing.DeploymentNotFound.URN()),
			fault.Internal("no deployment found for ID or wrong environment"),
			fault.Public("Deployment not found"),
		)
	}

	if deployment.EnvironmentID != s.environmentID {
		logger.Warn("deployment does not belong to this environment",
			"deploymentID", deploymentID,
			"deploymentEnv", deployment.EnvironmentID,
			"sentinelEnv", s.environmentID,
		)

		sentinelDeploymentLookupTotal.WithLabelValues("not_found").Inc()
		// Return as not found to avoid leaking information about deployments in other environments
		return db.Deployment{}, fault.New("deployment not found",
			fault.Code(codes.Sentinel.Routing.DeploymentNotFound.URN()),
			fault.Internal(fmt.Sprintf("deployment %s belongs to environment %s, but sentinel serves %s", deploymentID, deployment.EnvironmentID, s.environmentID)),
			fault.Public("Deployment not found"),
		)
	}

	return deployment, nil
}
