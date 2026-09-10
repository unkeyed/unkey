// Package readiness holds the healthy-regions rule Deploy and Wake both wait on.
package readiness

import (
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

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

	healthyRegions := 0
	for regionID, minReplicas := range regionMinReplicas {
		if runningPerRegion[regionID] >= minReplicas {
			healthyRegions++
		}
	}

	logger.Info(
		"checked instances",
		"deployment_id", deploymentID,
		"healthy_regions", healthyRegions,
		"required_regions", requiredRegions,
	)
	return healthyRegions >= requiredRegions, nil
}
