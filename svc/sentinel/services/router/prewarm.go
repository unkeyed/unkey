package router

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
)

func (s *service) prewarm(ctx context.Context) {
	logger.Info("prewarming cache")

	deployments, err := db.Query.ListDeploymentsByEnvironmentIdAndStatus(ctx, s.db.RO(), db.ListDeploymentsByEnvironmentIdAndStatusParams{
		EnvironmentID: s.environmentID,
		Status:        db.DeploymentsStatusReady,
		CreatedBefore: time.Now().UnixMilli(),
		UpdatedBefore: sql.NullInt64{Valid: false, Int64: 0},
	})
	if err != nil {
		logger.Error("unable to prewarm deployment cache", "error", err.Error())
		return
	}

	region, err := db.Query.FindRegionByPlatformAndName(ctx, s.db.RO(), db.FindRegionByPlatformAndNameParams{
		Platform: s.platform,
		Name:     s.region,
	})
	if err != nil {
		logger.Error("unable to find region for prewarming instance cache", "platform", s.platform, "region", s.region, "error", err.Error())
		return
	}

	for _, d := range deployments {
		instances, err := db.Query.FindInstancesByDeploymentIdAndRegionID(ctx, s.db.RO(), db.FindInstancesByDeploymentIdAndRegionIDParams{
			DeploymentID: d.ID,
			RegionID:     region.ID,
		})
		if err != nil {
			logger.Error("unable to find instances for deployment", "deployment_id", d.ID, "error", err.Error())
			continue
		}

		logger.Info("precaching deployment", "deployment_id", d.ID)
		s.deploymentCache.Set(ctx, d.ID, d)
		s.instancesCache.Set(ctx, d.ID, instances)

		policies, parseErr := engine.ParseMiddleware(d.SentinelConfig)
		if parseErr != nil {
			logger.Error("unable to parse sentinel config for deployment", "deployment_id", d.ID, "error", parseErr.Error())
		} else if policies != nil {
			s.policyCache.Set(ctx, d.ID, policies)
		}
	}

	logger.Info("deployment and instance cache are warm")
}
