package reconciler

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// refreshCurrentDeployments periodically reconciles all deployment ReplicaSets with
// their desired state from the control plane.
//
// This function runs every minute as a consistency safety net. While
// [Reconciler.watchCurrentDeployments] handles real-time events, watches can miss
// events during network partitions, controller restarts, or if the watch channel
// buffer overflows. This refresh loop guarantees eventual consistency by querying
// the control plane for each existing ReplicaSet and applying any needed changes.
//
// The function paginates through all krane-managed ReplicaSets across all namespaces
// to handle clusters with large numbers of deployments.
func (r *Reconciler) refreshCurrentDeployments(ctx context.Context) {
	repeat.Every(1*time.Minute, func() {
		r.logger.Info("refreshing current deployments")

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
