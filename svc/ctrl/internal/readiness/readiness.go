// Package readiness holds the healthy-regions rule Deploy and Wake both wait on.
package readiness

import (
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// RequiredRunningInstances returns how many running instances a region must
// have to count as healthy for a declared minimum replica count.
//
// Deployments always run at least one replica per region:
// [db.InsertDeploymentTopologyParams] rows are written with an
// autoscaling_replicas_min of at least 1, and krane floors the HPA minimum to
// the same value. A stored minimum of 0 therefore cannot mean "zero instances
// is healthy", and this floor keeps the readiness rule from passing a region
// with nothing running.
func RequiredRunningInstances(minReplicas uint32) uint32 {
	return max(minReplicas, 1)
}

// HealthyRegions counts the regions that run at least
// [RequiredRunningInstances] instances for their declared minimum.
func HealthyRegions(runningPerRegion map[string]uint32, regionMinReplicas map[string]uint32) int {
	healthy := 0
	for regionID, minReplicas := range regionMinReplicas {
		if runningPerRegion[regionID] >= RequiredRunningInstances(minReplicas) {
			healthy++
		}
	}
	return healthy
}

// InstancesHealthy reports whether at least requiredRegions regions run their
// minimum replica count
func InstancesHealthy(
	ctx context.Context,
	database db.Database,
	deploymentID string,
	regionMinReplicas map[string]uint32,
	requiredRegions int,
) (bool, error) {
	instances, err := database.FindInstancesByDeploymentId(ctx, deploymentID)
	if err != nil {
		return false, err
	}

	runningPerRegion := make(map[string]uint32)
	for _, instance := range instances {
		if instance.Status == db.InstancesStatusRunning {
			runningPerRegion[instance.RegionID]++
		}
	}

	healthyRegions := HealthyRegions(runningPerRegion, regionMinReplicas)

	logger.Info(
		"checked instances",
		"deployment_id", deploymentID,
		"healthy_regions", healthyRegions,
		"required_regions", requiredRegions,
	)
	return healthyRegions >= requiredRegions, nil
}
