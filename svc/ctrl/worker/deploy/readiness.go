package deploy

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/readiness"
)

const instancesReadyPromise = "instances_ready"

// waitForDeployments blocks until enough regions are healthy, or
// [regionReadyTimeout] elapses. A region is healthy when it has at least
// autoscaling_replicas_min running instances; the check tolerates one full
// regional outage by requiring (numRegions - 1) healthy regions, minimum 1.
//
// After one DB check for instances that are already healthy, the run awaits
// the [instancesReadyPromise], which services/cluster resolves through
// [Workflow.NotifyInstancesReady] once krane reports enough running instances.
// A resolve that lands before this point is kept by the promise, so there is
// nothing to register or clear.
func (w *Workflow) waitForDeployments(ctx restate.WorkflowContext, deploymentID string, topologies []db.InsertDeploymentTopologyParams) error {
	regionMinReplicas := make(map[string]uint32, len(topologies))
	for _, topo := range topologies {
		regionMinReplicas[topo.RegionID] = topo.AutoscalingReplicasMin
	}
	requiredRegions := max(len(regionMinReplicas)-1, 1)

	logger.Info(
		"waiting for deployments to be ready",
		"deployment_id", deploymentID,
		"total_regions", len(regionMinReplicas),
		"required_regions", requiredRegions,
	)

	alreadyHealthy, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return readiness.InstancesHealthy(runCtx, w.db, deploymentID, regionMinReplicas, requiredRegions)
	}, restate.WithName("initial healthy-regions check"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		logger.Warn("initial healthy-regions check failed, awaiting NotifyInstancesReady", "deployment_id", deploymentID, "error", err)
		alreadyHealthy = false
	}
	if alreadyHealthy {
		logger.Info("deployments ready", "deployment_id", deploymentID)
		return nil
	}

	ready := restate.Promise[restate.Void](ctx, instancesReadyPromise)
	timeout := restate.After(ctx, regionReadyTimeout)
	winner, err := restate.WaitFirst(ctx, ready, timeout)
	if err != nil {
		return fmt.Errorf("wait for healthy regions or timeout: %w", err)
	}
	if winner == ready {
		if _, err := ready.Result(); err != nil {
			return fmt.Errorf("instances ready promise: %w", err)
		}
		logger.Info("deployments ready", "deployment_id", deploymentID)
		return nil
	}

	return fault.Wrap(
		restate.TerminalErrorf("not enough regions became healthy in %v, required %d of %d", regionReadyTimeout, requiredRegions, len(regionMinReplicas)),
		fault.Public("Not enough regions became healthy in time."),
	)
}
