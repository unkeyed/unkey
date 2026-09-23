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

type readinessResult string

const (
	readinessReady              readinessResult = "ready"
	readinessOutOfMemory        readinessResult = "out_of_memory"
	readinessCrashLoopBackOff   readinessResult = "crash_loop_back_off"
	readinessContainerCannotRun readinessResult = "container_cannot_run"
	readinessInvalidImageName   readinessResult = "invalid_image_name"
)

// waitForDeployments blocks until enough regions are healthy, a startup failure
// is reported, or [regionReadyTimeout] elapses. A region is healthy when it has at least
// autoscaling_replicas_min running instances; the check tolerates one full
// regional outage by requiring (numRegions - 1) healthy regions, minimum 1.
//
// After one DB check for instances that are already healthy, the run awaits
// the [instancesReadyPromise], which services/cluster completes through
// [Workflow.NotifyReadiness] once krane reports enough running instances or
// a startup failure. Success resolves the promise; failures reject it so older
// runs awaiting restate.Void cannot mistake a failure for readiness. The first
// completion wins, including when it arrives before the wait starts.
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
		logger.Warn("initial healthy-regions check failed, awaiting NotifyReadiness", "deployment_id", deploymentID, "error", err)
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
		return fault.Wrap(fmt.Errorf("wait for healthy regions or timeout: %w", err), fault.Public("Instances did not become healthy in time."))
	}
	if winner == ready {
		_, err := ready.Result()
		if err != nil {
			return readinessResult(err.Message()).err()
		}
		return nil
	}

	return fault.Wrap(
		restate.TerminalErrorf("not enough regions became healthy in %v, required %d of %d", regionReadyTimeout, requiredRegions, len(regionMinReplicas)),
		fault.Public("Not enough regions became healthy in time."),
	)
}

func (r readinessResult) err() error {
	switch r {
	case readinessReady:
		return nil
	case readinessOutOfMemory:
		return fault.Wrap(
			restate.TerminalErrorf("instance OOMKilled during startup"),
			fault.Public("Your app ran out of memory during startup. Increase the memory limit or reduce startup memory use, then redeploy."),
		)
	case readinessCrashLoopBackOff:
		return fault.Wrap(
			restate.TerminalErrorf("instance CrashLoopBackOff during startup"),
			fault.Public("Your app repeatedly exited during startup. Check the runtime logs and start command, then redeploy."),
		)
	case readinessContainerCannotRun:
		return fault.Wrap(
			restate.TerminalErrorf("instance ContainerCannotRun during startup"),
			fault.Public("Your app could not start. Check the image's entrypoint, executable permissions, and CPU architecture, then redeploy."),
		)
	case readinessInvalidImageName:
		return fault.Wrap(
			restate.TerminalErrorf("instance InvalidImageName during startup"),
			fault.Public("The container image reference is invalid. Check the image name and tag or digest, then redeploy."),
		)
	}
	return fault.Wrap(restate.TerminalErrorf("invalid readiness result %q", r), fault.Public("Could not check deployment readiness."))
}
