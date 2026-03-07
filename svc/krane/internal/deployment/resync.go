package deployment

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// runResyncLoop periodically reconciles all Deployments with their desired state
// from the control plane.
//
// The loop runs every minute as a consistency safety net. While
// [Controller.runActualStateReportLoop] handles real-time Kubernetes events and
// [Controller.runDesiredStateApplyLoop] handles streaming updates, both can miss
// events during network partitions, controller restarts, or watch buffer overflows.
// This resync loop guarantees eventual consistency by querying the control plane
// for each existing Deployment and applying any drift.
//
// The loop paginates through all krane-managed Deployments across all namespaces,
// calling GetDesiredDeploymentState for each and applying or deleting as directed.
// Errors are logged but don't stop the loop from processing remaining Deployments.
//
// After reconciling Deployments, the loop cleans up orphan ReplicaSets that were
// created by the previous ReplicaSet-based controller and are not owned by any
// Deployment.
func (c *Controller) runResyncLoop(ctx context.Context) {
	repeat.Every(1*time.Minute, func() {
		logger.Info("running periodic resync")

		cursor := ""
		for {
			deployments, err := c.clientSet.AppsV1().Deployments("").List(ctx, metav1.ListOptions{
				LabelSelector: labels.New().
					ManagedByKrane().
					ComponentDeployment().
					ToString(),
				Continue: cursor,
			})
			if err != nil {
				logger.Error("unable to list deployments", "error", err.Error())
				return
			}

			for _, dep := range deployments.Items {
				deploymentID, ok := labels.GetDeploymentID(dep.Labels)
				if !ok {
					logger.Error("unable to get deployment ID", "deployment", dep.Name)
					continue
				}

				res, err := c.cluster.GetDesiredDeploymentState(ctx, &ctrlv1.GetDesiredDeploymentStateRequest{
					DeploymentId: deploymentID,
				})
				if err != nil {
					if connect.CodeOf(err) == connect.CodeNotFound {
						if err := c.DeleteDeployment(ctx, &ctrlv1.DeleteDeployment{
							K8SNamespace: dep.GetNamespace(),
							K8SName:      dep.GetName(),
						}); err != nil {
							logger.Error("unable to delete deployment", "error", err.Error(), "deployment_id", deploymentID)
							continue
						}
					}

					logger.Error("unable to get desired deployment state", "error", err.Error(), "deployment_id", deploymentID)
					continue
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

			cursor = deployments.Continue
			if cursor == "" {
				break
			}
		}

		// Clean up orphan ReplicaSets from the previous controller that are not
		// owned by a Deployment. These are standalone RS with krane labels.
		c.cleanupOrphanReplicaSets(ctx)
	})
}

// cleanupOrphanReplicaSets deletes standalone ReplicaSets with krane labels that
// have no ownerReferences (i.e. not managed by a Deployment). These are leftovers
// from before we switched customer workloads from ReplicaSets to Deployments.
func (c *Controller) cleanupOrphanReplicaSets(ctx context.Context) {
	cursor := ""
	for {
		rsList, err := c.clientSet.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{
			LabelSelector: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				ToString(),
			Continue: cursor,
		})
		if err != nil {
			logger.Error("unable to list orphan replicasets", "error", err.Error())
			return
		}

		for _, rs := range rsList.Items {
			// Skip RS owned by a Deployment (created by the new controller)
			if len(rs.OwnerReferences) > 0 {
				continue
			}
			logger.Info("deleting orphan replicaset", "namespace", rs.Namespace, "name", rs.Name)
			if err := c.clientSet.AppsV1().ReplicaSets(rs.Namespace).Delete(ctx, rs.Name, metav1.DeleteOptions{}); err != nil {
				logger.Error("unable to delete orphan replicaset", "error", err.Error(), "name", rs.Name)
			}
		}

		cursor = rsList.Continue
		if cursor == "" {
			break
		}
	}
}
