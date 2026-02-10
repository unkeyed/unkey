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
// [Controller.runActualStateReportLoop] handles real-time Kubernetes events and
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
				deploymentID, ok := labels.GetDeploymentID(replicaSet.Labels)
				if !ok {
					logger.Error("unable to get deployment ID", "replicaSet", replicaSet.Name)
					continue
				}

				res, err := c.cluster.GetDesiredDeploymentState(ctx, connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{
					DeploymentId: deploymentID,
				}))
				if err != nil {
					logger.Error("unable to get desired deployment state", "error", err.Error(), "deployment_id", deploymentID)
					continue
				}

				switch res.Msg.GetState().(type) {
				case *ctrlv1.DeploymentState_Apply:
					if err := c.ApplyDeployment(ctx, res.Msg.GetApply()); err != nil {
						logger.Error("unable to apply deployment", "error", err.Error(), "deployment_id", deploymentID)
					}
				case *ctrlv1.DeploymentState_Delete:
					if err := c.DeleteDeployment(ctx, res.Msg.GetDelete()); err != nil {
						logger.Error("unable to delete deployment", "error", err.Error(), "deployment_id", deploymentID)
					}
				}
			}

			cursor = replicaSets.Continue
			if cursor == "" {
				break
			}
		}
	})
}
