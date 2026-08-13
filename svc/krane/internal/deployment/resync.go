package deployment

import (
	"context"
	"fmt"
	"sort"
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

// runActualStateResyncLoop periodically reports actual instance state to the
// control plane for every deployment ReplicaSet.
//
// This is a fast, lightweight safety net that complements [Controller.runPodWatchLoop].
// The watch handles real-time events, but can miss updates during network partitions,
// restarts, or buffer overflows. This loop catches the drift by rebuilding and
// reporting status for every RS every 30 seconds.
//
// This loop does NOT fetch or apply desired state — that is handled independently
// by [Controller.runDesiredStateResyncLoop] so that slow control plane RPCs cannot
// delay instance reporting.
func (c *Controller) runActualStateResyncLoop(ctx context.Context) {
	repeat.Every(30*time.Second, func() {
		logger.Info("running actual state resync")
		c.resyncActualState(ctx)
	})
}

func (c *Controller) resyncActualState(ctx context.Context) {
	replicaSets, err := c.reportDeploymentInventory(ctx)
	if err != nil {
		logger.Error("actual state resync: unable to report complete deployment inventory", "error", err.Error())
	}

	// A partial list is still useful for refreshing observed pod state. It is
	// only unsafe as an authoritative inventory, which was skipped above.
	conc.ForEach(ctx, replicaSets, func(ctx context.Context, rs *appsv1.ReplicaSet) {
		status, err := c.buildDeploymentStatus(ctx, rs)
		if err != nil {
			logger.Error("actual state resync: unable to build deployment status", "error", err.Error(), "replicaSet", rs.Name)
			return
		}
		reported, err := c.reportIfChanged(ctx, status)
		if err != nil {
			logger.Error("actual state resync: unable to report deployment status", "error", err.Error(), "replicaSet", rs.Name)
			return
		}
		if reported {
			// Resync found drift the watch didn't deliver. This is the
			// "pod watch missed an event" smoking-gun signal — a
			// healthy cluster should see this counter stay flat.
			metrics.ResyncCorrectionsTotal.WithLabelValues("deployment").Inc()
			logger.Info("actual state resync: reported changed deployment status", "replicaSet", rs.Name)
		}
	})
}

func (c *Controller) reportDeploymentInventory(ctx context.Context) ([]appsv1.ReplicaSet, error) {
	// Hold status reports until the complete inventory is accepted. Otherwise
	// a watch event for a ReplicaSet created during the list could be upserted
	// first and then incorrectly removed by the older inventory.
	c.statusReportMu.Lock()
	defer c.statusReportMu.Unlock()

	replicaSets, err := c.listReplicaSets(ctx)
	if err != nil {
		return replicaSets, err
	}

	deploymentIDs := make([]string, 0, len(replicaSets))
	for i := range replicaSets {
		deploymentID, ok := labels.GetDeploymentID(replicaSets[i].Labels)
		if !ok {
			return replicaSets, fmt.Errorf("replicaSet %s is missing deployment ID", replicaSets[i].Name)
		}
		deploymentIDs = append(deploymentIDs, deploymentID)
	}
	sort.Strings(deploymentIDs)
	err = c.sendDeploymentStatus(ctx, &ctrlv1.ReportDeploymentStatusRequest{
		Change: &ctrlv1.ReportDeploymentStatusRequest_Inventory_{
			Inventory: &ctrlv1.ReportDeploymentStatusRequest_Inventory{
				DeploymentIds: deploymentIDs,
			},
		},
	})
	return replicaSets, err
}

// runDesiredStateResyncLoop periodically reconciles every deployment ReplicaSet
// against the control plane's desired state.
//
// This is a consistency safety net that complements the streaming desired state
// channel. It runs every minute, fetching the desired state for each RS and
// applying or deleting as needed. Because this involves potentially slow RPCs
// (GetDesiredDeploymentState), it runs independently from actual state reporting
// so it cannot delay instance updates.
func (c *Controller) runDesiredStateResyncLoop(ctx context.Context) {
	repeat.Every(1*time.Minute, func() {
		logger.Info("running desired state resync")
		replicaSets, err := c.listReplicaSets(ctx)
		if err != nil {
			logger.Error("desired state resync: unable to list all replicaSets", "error", err.Error())
		}
		conc.ForEach(ctx, replicaSets, func(ctx context.Context, rs *appsv1.ReplicaSet) {
			c.reconcileDesiredState(ctx, rs)
		})
	})
}

// listReplicaSets paginates through all krane-managed deployment ReplicaSets.
// On a later-page failure it returns the partial list and an error, allowing
// callers to process known resources without treating the list as authoritative.
func (c *Controller) listReplicaSets(ctx context.Context) ([]appsv1.ReplicaSet, error) {
	replicaSets := []appsv1.ReplicaSet{}
	cursor := ""
	for {
		page, err := c.clientSet.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{
			LabelSelector: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				ToString(),
			Continue: cursor,
		})
		if err != nil {
			return replicaSets, err
		}

		replicaSets = append(replicaSets, page.Items...)

		cursor = page.Continue
		if cursor == "" {
			return replicaSets, nil
		}
	}
}

// reconcileDesiredState fetches the desired state for a single ReplicaSet from
// the control plane and applies or deletes as needed.
func (c *Controller) reconcileDesiredState(ctx context.Context, replicaSet *appsv1.ReplicaSet) {
	deploymentID, ok := labels.GetDeploymentID(replicaSet.Labels)
	if !ok {
		logger.Error("unable to get deployment ID", "replicaSet", replicaSet.Name)
		return
	}

	res, err := c.cluster.GetDesiredDeploymentState(ctx, &ctrlv1.GetDesiredDeploymentStateRequest{
		Cluster:      c.clusterKey(),
		DeploymentId: deploymentID,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			if err := c.DeleteDeployment(ctx, &ctrlv1.DeleteDeployment{
				K8SNamespace: replicaSet.GetNamespace(),
				K8SName:      replicaSet.GetName(),
			}); err != nil {
				logger.Error("unable to delete deployment", "error", err.Error(), "deployment_id", deploymentID)
			}

			return
		}

		logger.Error("unable to get desired deployment state", "error", err.Error(), "deployment_id", deploymentID)
		return
	}

	switch res.GetState().(type) {
	case *ctrlv1.DeploymentState_Apply:
		if err := c.ApplyDeployment(ctx, res.GetApply()); err != nil {
			logger.Error("unable to apply deployment", "error", err.Error(), "deployment_id", deploymentID)
		}
	case *ctrlv1.DeploymentState_Delete:
		if err := c.DeleteDeployment(ctx, res.GetDelete()); err != nil {
			logger.Error("unable to delete deployment", "error", err.Error(), "deployment_id", deploymentID)
		}
	}
}
