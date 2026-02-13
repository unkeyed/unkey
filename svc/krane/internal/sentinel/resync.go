package sentinel

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

// runResyncLoop periodically reconciles all sentinel Deployments with their
// desired state from the control plane.
//
// This loop runs every minute as a consistency safety net. While
// [Controller.runActualStateReportLoop] handles real-time K8s events and
// [Controller.runDesiredStateApplyLoop] handles streaming updates, both can miss
// events during network partitions, controller restarts, or buffer overflows.
// This resync loop guarantees eventual consistency by querying the control plane
// for each existing sentinel and applying any needed changes.
func (c *Controller) runResyncLoop(ctx context.Context) {
	repeat.Every(1*time.Minute, nil, func() {
		logger.Info("running periodic resync")

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

			for _, deployment := range deployments.Items {
				sentinelID, ok := labels.GetSentinelID(deployment.Labels)
				if !ok {
					logger.Error("unable to get sentinel ID", "deployment", deployment.Name)
					continue
				}

				res, err := c.cluster.GetDesiredSentinelState(ctx, connect.NewRequest(&ctrlv1.GetDesiredSentinelStateRequest{
					SentinelId: sentinelID,
				}))
				if err != nil {
					if connect.CodeOf(err) == connect.CodeNotFound {
						if err := c.DeleteSentinel(ctx, &ctrlv1.DeleteSentinel{
							K8SName: deployment.GetName(),
						}); err != nil {
							logger.Error("unable to delete sentinel", "error", err.Error(), "sentinel_id", sentinelID)
							continue
						}
					}

					logger.Error("unable to get desired sentinel state", "error", err.Error(), "sentinel_id", sentinelID)
					continue
				}

				switch res.Msg.GetState().(type) {
				case *ctrlv1.SentinelState_Apply:
					if err := c.ApplySentinel(ctx, res.Msg.GetApply()); err != nil {
						logger.Error("unable to apply sentinel", "error", err.Error(), "sentinel_id", sentinelID)
					}
				case *ctrlv1.SentinelState_Delete:
					if err := c.DeleteSentinel(ctx, res.Msg.GetDelete()); err != nil {
						logger.Error("unable to delete sentinel", "error", err.Error(), "sentinel_id", sentinelID)
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
