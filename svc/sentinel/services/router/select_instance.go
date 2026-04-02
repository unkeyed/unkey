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
	"github.com/unkeyed/unkey/svc/sentinel/services/balancer"
)

func (s *service) SelectInstance(ctx context.Context, deploymentID string) (db.Instance, error) {
	t := time.Now()

	instances, hit, err := s.instancesCache.SWR(ctx, deploymentID, func(ctx context.Context) ([]db.Instance, error) {

		region, err := db.Query.FindRegionByPlatformAndName(ctx, s.db.RO(), db.FindRegionByPlatformAndNameParams{
			Platform: s.platform,
			Name:     s.region,
		})
		if err != nil {
			return nil, err
		}

		return db.Query.FindInstancesByDeploymentIdAndRegionID(
			ctx,
			s.db.RO(),
			db.FindInstancesByDeploymentIdAndRegionIDParams{
				DeploymentID: deploymentID,
				RegionID:     region.ID,
			},
		)
	}, caches.DefaultFindFirstOp)

	sentinelRoutingDuration.WithLabelValues("select_instance").Observe(time.Since(t).Seconds())

	if err != nil {
		sentinelInstanceSelectionTotal.WithLabelValues("error").Inc()
		return db.Instance{}, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("failed to get instances"),
		)
	}

	if hit == cache.Null || len(instances) == 0 {
		sentinelInstanceSelectionTotal.WithLabelValues("no_instances").Inc()
		return db.Instance{}, fault.New("no instances found",
			fault.Code(codes.Sentinel.Routing.NoRunningInstances.URN()),
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
		sentinelInstanceSelectionTotal.WithLabelValues("no_running_instances").Inc()
		return db.Instance{}, fault.New("no running instances",
			fault.Code(codes.Sentinel.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no running instances for deployment %s in region %s (found %d total)", deploymentID, s.region, len(instances))),
			fault.Public("Service temporarily unavailable"),
		)
	}

	sentinelInstanceSelectionTotal.WithLabelValues("success").Inc()

	ids := make([]string, len(runningInstances))
	for i, inst := range runningInstances {
		ids[i] = inst.ID
	}
	idx := s.balancer.Pick(ids)
	selected := runningInstances[idx]
	if tracker, ok := s.balancer.(balancer.InflightTracker); ok {
		tracker.Acquire(selected.ID)
	}
	return selected, nil
}

func (s *service) ReleaseInstance(instanceID string) {
	if tracker, ok := s.balancer.(balancer.InflightTracker); ok {
		tracker.Release(instanceID)
	}
}
