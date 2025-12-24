package sentinel

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/labels"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/repeat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// refreshCurrentdeployments performs periodic synchronization of all deployment resources.
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

			deployments, err := r.clientSet.AppsV1().Deployments("").List(ctx, metav1.ListOptions{
				LabelSelector: labels.New().
					ManagedByKrane().
					ComponentSentinel().
					ToString(),
				Continue: cursor,
			})

			if err != nil {
				r.logger.Error("unable to list deployments", "error", err.Error())
				return
			}

			for _, deployment := range deployments.Items {

				sentinelID, ok := labels.GetSentinelID(deployment.Labels)
				if !ok {
					r.logger.Error("unable to get sentinel ID", "error", "deployment", deployment)
					continue
				}

				res, err := r.cluster.GetDesiredSentinelState(ctx, connect.NewRequest(&ctrlv1.GetDesiredSentinelStateRequest{
					SentinelId: sentinelID,
				}))
				if err != nil {
					r.logger.Error("unable to get desired sentinel state", "error", err.Error(), "sentinel_id", sentinelID)
					continue
				}

				r.inbound.Buffer(res.Msg)
			}
			cursor = deployments.Continue
			if cursor == "" {
				break
			}
		}

	})
}
