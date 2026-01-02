package reconciler

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/apps/krane/pkg/labels"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/repeat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// refreshCurrentReplicaSets performs periodic synchronization of all deployment resources.
//
// This function runs every minute to ensure all deployment resources in the
// cluster are synchronized with their desired state from the control plane.
// This periodic refresh provides consistency guarantees despite possible
// missed events, network partitions, or controller restarts.
//
// The function:
//  1. Lists all deployment resources managed by krane across all namespaces
//  2. Queries control plane for the desired state of each deployment
//  3. Buffers the desired state events for processing
//
// This approach ensures eventual consistency between the database state
// and Kubernetes cluster state, acting as a safety net for the event-based
// synchronization mechanism.
func (r *Reconciler) refreshCurrentDeployments(ctx context.Context) {
	repeat.Every(1*time.Minute, func() {

		cursor := ""
		for {

			replicaSets, err := r.clientSet.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{
				LabelSelector: labels.New().
					ManagedByKrane().
					ComponentDeployment().
					ToString(),
				Continue: cursor,
			})

			if err != nil {
				r.logger.Error("unable to list replicaSets", "error", err.Error())
				return
			}

			for _, replicaSet := range replicaSets.Items {

				deploymentID, ok := labels.GetDeploymentID(replicaSet.Labels)
				if !ok {
					r.logger.Error("unable to get deployment ID", "error", "replicaSet", replicaSet)
					continue
				}

				res, err := r.cluster.GetDesiredDeploymentState(ctx, connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{
					DeploymentId: deploymentID,
				}))
				if err != nil {
					r.logger.Error("unable to get desired deployment state", "error", err.Error(), "deployment_id", deploymentID)
					continue
				}

				switch res.Msg.GetState().(type) {
				case *ctrlv1.DeploymentState_Apply:
					if err := r.ApplyDeployment(ctx, res.Msg.GetApply()); err != nil {
						r.logger.Error("unable to apply deployment", "error", err.Error(), "deployment_id", deploymentID)
					}
				case *ctrlv1.DeploymentState_Delete:
					if err := r.DeleteDeployment(ctx, res.Msg.GetDelete()); err != nil {
						r.logger.Error("unable to delete deployment", "error", err.Error(), "deployment_id", deploymentID)
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
