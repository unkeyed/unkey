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

// runResyncLoop periodically reconciles all deployment ReplicaSets with their
// desired state from the control plane.
//
// The loop runs every minute as a consistency safety net. While
// [Controller.runPodWatchLoop] handles real-time Kubernetes events and
// [Controller.runDesiredStateApplyLoop] handles streaming updates, both can miss
// events during network partitions, controller restarts, or watch buffer overflows.
// This resync loop guarantees eventual consistency by querying the control plane
// for each existing ReplicaSet and applying any drift.
//
// The loop paginates through all krane-managed deployment ReplicaSets across all
// namespaces, calling GetDesiredDeploymentState for each and applying or deleting
// as directed. Errors are logged but don't stop the loop from processing remaining
// ReplicaSets.
func (c *Controller) runResyncLoop(ctx context.Context) {
	repeat.Every(1*time.Minute, func() {
		logger.Info("running periodic resync")

		cursor := ""
		for {
			replicaSets, err := c.clientSet.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{
				LabelSelector: labels.New().
					ManagedByKrane().
					ComponentDeployment().
					ToString(),
				Continue: cursor,
			})
			if err != nil {
				logger.Error("unable to list replicaSets", "error", err.Error())
				return
			}

			for _, replicaSet := range replicaSets.Items {
				// Report actual state back to the control plane if it differs
				// from the last report. This catches pods that were skipped
				// earlier (e.g. no IP yet) or watch events that were missed.
				status, buildErr := c.buildDeploymentStatus(ctx, &replicaSet)
				if buildErr != nil {
					logger.Error("resync: unable to build deployment status", "error", buildErr.Error(), "replicaSet", replicaSet.Name)
				} else if reported, reportErr := c.reportIfChanged(ctx, status); reportErr != nil {
					logger.Error("resync: unable to report deployment status", "error", reportErr.Error(), "replicaSet", replicaSet.Name)
				} else if reported {
					logger.Info("resync: reported changed deployment status", "replicaSet", replicaSet.Name)
				}

				deploymentID, ok := labels.GetDeploymentID(replicaSet.Labels)
				if !ok {
					logger.Error("unable to get deployment ID", "replicaSet", replicaSet.Name)
					continue
				}

				res, err := c.cluster.GetDesiredDeploymentState(ctx, &ctrlv1.GetDesiredDeploymentStateRequest{
					DeploymentId: deploymentID,
				})
				if err != nil {
					if connect.CodeOf(err) == connect.CodeNotFound {
						if err := c.DeleteDeployment(ctx, &ctrlv1.DeleteDeployment{
							K8SNamespace: replicaSet.GetNamespace(),
							K8SName:      replicaSet.GetName(),
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

			cursor = replicaSets.Continue
			if cursor == "" {
				break
			}
		}
	}, nil)
}
