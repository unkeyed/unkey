package deployment

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
)

// runActualStateResyncLoop periodically reports actual instance state to the
// control plane for every workload.
//
// This is a fast, lightweight safety net that complements [Controller.runPodWatchLoop].
// The watch handles real-time events, but can miss updates during network partitions,
// restarts, or buffer overflows. This loop catches the drift by rebuilding and
// reporting status for every workload every 30 seconds.
//
// This loop does NOT fetch or apply desired state — that is handled independently
// by [Controller.runDesiredStateResyncLoop] so that slow control plane RPCs cannot
// delay instance reporting.
func (c *Controller) runActualStateResyncLoop(ctx context.Context) {
	c.runResyncLoop(ctx, 30*time.Second, func() {
		logger.Info("running actual state resync")
		c.forEachWorkload(ctx, func(ctx context.Context, w workload) {
			status, err := c.buildDeploymentStatus(ctx, w)
			if err != nil {
				logger.Error("actual state resync: unable to build deployment status", "error", err.Error(), "workload", w.k8sName)
				return
			}
			reported, err := c.reportIfChanged(ctx, status)
			if err != nil {
				logger.Error("actual state resync: unable to report deployment status", "error", err.Error(), "workload", w.k8sName)
				return
			}
			if reported {
				// Resync found drift the watch didn't deliver. This is the
				// "pod watch missed an event" smoking-gun signal — a
				// healthy cluster should see this counter stay flat.
				metrics.ResyncCorrectionsTotal.WithLabelValues("deployment").Inc()
				logger.Info("actual state resync: reported changed deployment status", "workload", w.k8sName)
			}
		})
	})
}

// runDesiredStateResyncLoop periodically reconciles every workload against the
// control plane's desired state.
//
// This is a consistency safety net that complements the streaming desired state
// channel. It runs every minute, fetching the desired state for each workload and
// applying or deleting as needed. Because this involves potentially slow RPCs
// (GetDesiredDeploymentState), it runs independently from actual state reporting
// so it cannot delay instance updates.
func (c *Controller) runDesiredStateResyncLoop(ctx context.Context) {
	c.runResyncLoop(ctx, time.Minute, func() {
		logger.Info("running desired state resync")
		c.forEachWorkload(ctx, c.reconcileDesiredState)
	})
}

func (c *Controller) runResyncLoop(ctx context.Context, interval time.Duration, resync func()) {
	resync()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resync()
		}
	}
}

// reconcileDesiredState fetches the desired state for a single workload from
// the control plane and applies or deletes as needed.
func (c *Controller) reconcileDesiredState(ctx context.Context, w workload) {
	res, err := c.cluster.GetDesiredDeploymentState(ctx, &ctrlv1.GetDesiredDeploymentStateRequest{
		Cluster:      c.clusterKey(),
		DeploymentId: w.deploymentID,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			if err := c.DeleteDeployment(ctx, &ctrlv1.DeleteDeployment{
				K8SNamespace: w.namespace,
				K8SName:      w.k8sName,
			}); err != nil {
				logger.Error("unable to delete deployment", "error", err.Error(), "deployment_id", w.deploymentID)
			}

			return
		}

		logger.Error("unable to get desired deployment state", "error", err.Error(), "deployment_id", w.deploymentID)
		return
	}

	switch res.GetState().(type) {
	case *ctrlv1.DeploymentState_Apply:
		if err := c.ApplyDeployment(ctx, res.GetApply()); err != nil {
			logger.Error("unable to apply deployment", "error", err.Error(), "deployment_id", w.deploymentID)
		}
	case *ctrlv1.DeploymentState_Delete:
		if err := c.DeleteDeployment(ctx, res.GetDelete()); err != nil {
			logger.Error("unable to delete deployment", "error", err.Error(), "deployment_id", w.deploymentID)
		}
	}
}
