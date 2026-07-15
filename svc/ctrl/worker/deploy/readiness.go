package deploy

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/compensation"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// waitForDeployments blocks until enough regions are healthy, or
// [regionReadyTimeout] elapses. A region is considered healthy when it has
// at least autoscaling_replicas_min running instances. The check tolerates
// one full regional outage: it requires (numRegions - 1) healthy regions,
// minimum 1.
//
// The wait is push-based via a Restate awakeable. The handler:
//  1. Stores {awakeable_id, deployment_id} in VO state under
//     [instancesReadyAwakeableKey]
//  2. Does an initial DB check in case instances are already healthy (e.g.
//     a redeploy against already-running pods, or a report that landed
//     between createTopologies and here) and self-resolves the awakeable
//     if so
//  3. Races the awakeable against a [regionReadyTimeout] timeout via
//     [restate.WaitFirst]
//
// The awakeable is resolved by [Workflow.NotifyInstancesReady], which is
// called from services/cluster's ReportDeploymentStatus RPC whenever krane
// reports an instance status change that pushes the deployment past the
// healthy-region threshold.
//
// State cleanup: the caller passes `compensation` so the state clear can
// be registered as a durable compensation (survives Restate cancellation)
// rather than relying on a Go defer.
func (w *Workflow) waitForDeployments(ctx restate.ObjectContext, compensation *compensation.Compensation, deploymentID string, topologies []db.InsertDeploymentTopologyParams) error {
	// Build per-region minimum replica requirements.
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

	// Create awakeable and stash it in VO state BEFORE doing the initial
	// health check. This prevents a race where an instance report lands
	// between our check and the state write, causing NotifyInstancesReady
	// to find no awakeable and return a no-op.
	awk := restate.Awakeable[restate.Void](ctx)
	restate.Set(ctx, instancesReadyAwakeableKey, awk.Id())

	// Clear state on failure so the VO doesn't keep a stale awakeable_id
	// around after the deployment terminates.
	compensation.AddCtx(func(ctx restate.ObjectContext) error {
		restate.Clear(ctx, instancesReadyAwakeableKey)
		return nil
	})

	// Initial check: if instances are already healthy, resolve immediately.
	// Best-effort: on retry exhaustion, log and continue so NotifyInstancesReady
	// can still complete the wait via the awakeable.
	alreadyHealthy, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return w.checkInstancesHealthy(runCtx, deploymentID, regionMinReplicas, requiredRegions)
	}, restate.WithName("initial healthy-regions check"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		logger.Warn(
			"initial healthy-regions check failed, proceeding to await NotifyInstancesReady",
			"deployment_id", deploymentID,
			"error", err,
		)
		alreadyHealthy = false
	}
	if alreadyHealthy {
		restate.ResolveAwakeable[restate.Void](ctx, awk.Id(), restate.Void{})
	}

	// Race the awakeable against a timeout.
	timeout := restate.After(ctx, regionReadyTimeout)
	winner, err := restate.WaitFirst(ctx, awk, timeout)
	if err != nil {
		return fmt.Errorf("wait for healthy regions or timeout: %w", err)
	}

	if winner == awk {
		// Drain the result to surface any rejection error.
		if _, err := awk.Result(); err != nil {
			return fmt.Errorf("awakeable result: %w", err)
		}
		// Clear eagerly on the happy path. The compensation registered
		// above still runs on any later error in the Deploy workflow.
		restate.Clear(ctx, instancesReadyAwakeableKey)
		logger.Info("deployments ready", "deployment_id", deploymentID)
		return nil
	}

	return fault.Wrap(
		restate.TerminalErrorf("not enough regions became healthy in %v, required %d of %d", regionReadyTimeout, requiredRegions, len(regionMinReplicas)),
		fault.Public("Not enough regions became healthy in time."),
	)
}

// checkInstancesHealthy returns true if the current running-instance counts
// satisfy the per-region minimum replica requirements for at least
// requiredRegions. It is the same logic used by ReportDeploymentStatus in
// services/cluster to decide whether to call NotifyInstancesReady.
func (w *Workflow) checkInstancesHealthy(
	ctx context.Context,
	deploymentID string,
	regionMinReplicas map[string]uint32,
	requiredRegions int,
) (bool, error) {
	instances, err := w.db.FindInstancesByDeploymentId(ctx, deploymentID)
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
