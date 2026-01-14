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

// refreshCurrentSentinels periodically reconciles all sentinel Deployments with
// their desired state from the control plane.
//
// This function runs every minute as a consistency safety net. While
// [Reconciler.watchCurrentSentinels] handles real-time events, watches can miss
// events during network partitions, controller restarts, or if the watch channel
// buffer overflows. This refresh loop guarantees eventual consistency by querying
// the control plane for each existing sentinel and applying any needed changes.
//
// The function paginates through all krane-managed sentinel Deployments across all
// namespaces to handle clusters with large numbers of environments.
func (r *Reconciler) refreshCurrentSentinels(ctx context.Context) {

	repeat.Every(1*time.Minute, func() {

		cursor := ""
		for {

			deployments, err := r.clientSet.AppsV1().Deployments(SentinelNamespace).List(ctx, metav1.ListOptions{
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

				switch res.Msg.GetState().(type) {
				case *ctrlv1.SentinelState_Apply:
					if err := r.ApplySentinel(ctx, res.Msg.GetApply()); err != nil {
						r.logger.Error("unable to apply sentinel", "error", err.Error(), "sentinel_id", sentinelID)
					}
				case *ctrlv1.SentinelState_Delete:
					if err := r.DeleteSentinel(ctx, res.Msg.GetDelete()); err != nil {
						r.logger.Error("unable to delete sentinel", "error", err.Error(), "sentinel_id", sentinelID)
					}
				}
			}
			cursor = deployments.Continue
			if cursor == "" {
				break
			}
		}

	})
}
