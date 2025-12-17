package inbound

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/repeat"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *InboundReconciler) refreshCurrentSentinels(ctx context.Context, queue *buffer.Buffer[*ctrlv1.SentinelState]) {
	repeat.Every(1*time.Minute, func() {

		cursor := ""
		for {

			sentinels := sentinelv1.SentinelList{} // nolint:exhaustruct
			err := r.client.List(ctx, &sentinels, &client.ListOptions{
				LabelSelector: k8s.NewLabels().
					ManagedByKrane().
					ToSelector(),
				Continue:  cursor,
				Namespace: "", // empty to match across all
			})

			if err != nil {
				r.logger.Error("unable to list sentinels", "error", err.Error())
				return
			}

			for _, sentinel := range sentinels.Items {
				res, err := r.cluster.GetDesiredSentinelState(ctx, connect.NewRequest(&ctrlv1.GetDesiredSentinelStateRequest{
					SentinelId: sentinel.Spec.SentinelID,
				}))
				if err != nil {
					r.logger.Error("unable to get desired sentinel state", "error", err.Error(), "sentinel", sentinel)
					continue
				}

				queue.Buffer(res.Msg)
			}
			cursor = sentinels.Continue
			if cursor == "" {
				break
			}
		}

	})
}
