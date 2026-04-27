package sentinel

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// runActualStateResyncLoop periodically reports actual sentinel health to the
// control plane for every sentinel Deployment.
//
// This is a lightweight safety net that complements [Controller.runActualStateReportLoop].
// The watch handles real-time events, but can miss updates during network partitions,
// restarts, or buffer overflows. This loop catches the drift by reporting health
// for every sentinel every 30 seconds.
//
// This loop does NOT fetch or apply desired state — that is handled independently
// by [Controller.runDesiredStateResyncLoop] so that slow control plane RPCs cannot
// delay health reporting.
func (c *Controller) runActualStateResyncLoop(ctx context.Context) {
	repeat.Every(30*time.Second, func() {
		logger.Info("running sentinel actual state resync")
		c.forEachSentinelDeployment(ctx, func(ctx context.Context, deployment *appsv1.Deployment) {
			reported, err := c.reportSentinelState(ctx, deployment)
			if err != nil {
				logger.Error("actual state resync: unable to report sentinel status", "error", err.Error(), "name", deployment.Name)
				return
			}
			if reported {
				// Resync caught drift the real-time watch didn't deliver.
				// A healthy cluster should see this counter stay flat.
				metrics.ResyncCorrectionsTotal.WithLabelValues("sentinel").Inc()
				logger.Info("actual state resync: reported changed sentinel status", "name", deployment.Name)
			}
		})
	})
}

// runDesiredStateResyncLoop periodically reconciles every sentinel Deployment
// against the control plane's desired state.
//
// This is a consistency safety net that complements the streaming desired state
// channel. It runs every minute, fetching the desired state for each sentinel and
// applying or deleting as needed. Because this involves potentially slow RPCs
// (GetDesiredSentinelState), it runs independently from actual state reporting
// so it cannot delay health updates.
func (c *Controller) runDesiredStateResyncLoop(ctx context.Context) {
	repeat.Every(1*time.Minute, func() {
		logger.Info("running sentinel desired state resync")
		c.forEachSentinelDeployment(ctx, func(ctx context.Context, deployment *appsv1.Deployment) {
			c.reconcileDesiredState(ctx, deployment)
		})
	})
}

// forEachSentinelDeployment paginates through all krane-managed sentinel
// Deployments and calls fn for each one concurrently.
func (c *Controller) forEachSentinelDeployment(ctx context.Context, fn func(ctx context.Context, deployment *appsv1.Deployment)) {
	cursor := ""
	for {
		deployments, err := c.clientSet.AppsV1().Deployments(NamespaceSentinel).List(ctx, metav1.ListOptions{
			LabelSelector: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				ToString(),
			Continue: cursor,
		})
		if err != nil {
			logger.Error("unable to list deployments", "error", err.Error())
			return
		}

		conc.ForEach(ctx, deployments.Items, fn)

		cursor = deployments.Continue
		if cursor == "" {
			break
		}
	}
}

// reconcileDesiredState fetches the desired state for a single sentinel from
// the control plane and applies or deletes as needed.
func (c *Controller) reconcileDesiredState(ctx context.Context, deployment *appsv1.Deployment) {
	sentinelID, ok := labels.GetSentinelID(deployment.Labels)
	if !ok {
		logger.Error("unable to get sentinel ID", "deployment", deployment.Name)
		return
	}

	res, err := c.cluster.GetDesiredSentinelState(ctx, &ctrlv1.GetDesiredSentinelStateRequest{
		SentinelId: sentinelID,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			if err := c.DeleteSentinel(ctx, &ctrlv1.DeleteSentinel{
				K8SName: deployment.GetName(),
			}); err != nil {
				logger.Error("unable to delete sentinel", "error", err.Error(), "sentinel_id", sentinelID)
			}

			return
		}

		logger.Error("unable to get desired sentinel state", "error", err.Error(), "sentinel_id", sentinelID)
		return
	}

	switch res.GetState().(type) {
	case *ctrlv1.SentinelState_Apply:
		if err := c.ApplySentinel(ctx, res.GetApply()); err != nil {
			logger.Error("unable to apply sentinel", "error", err.Error(), "sentinel_id", sentinelID)
		}
	case *ctrlv1.SentinelState_Delete:
		if err := c.DeleteSentinel(ctx, res.GetDelete()); err != nil {
			logger.Error("unable to delete sentinel", "error", err.Error(), "sentinel_id", sentinelID)
		}
	}
}
