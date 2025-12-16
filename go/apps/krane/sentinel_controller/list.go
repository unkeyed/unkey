package sentinelcontroller

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetScheduledSentinelIDs returns a channel of all sentinel IDs managed by krane.
//
// This method lists all Sentinel CRDs across all namespaces and
// streams their sentinel IDs through the returned channel. The channel is
// closed when all sentinels have been processed.
//
// Parameters:
//   - ctx: Context for the listing operation
//
// Returns a channel that emits sentinel IDs. Errors are logged but
// do not affect channel operation.
func (c *SentinelController) GetScheduledSentinelIDs(ctx context.Context) <-chan string {
	sentinelIDs := make(chan string)

	go func() {

		defer close(sentinelIDs)

		cursor := ""
		for {

			sentinels := sentinelv1.SentinelList{} // nolint:exhaustruct
			err := c.client.List(ctx, &sentinels, &client.ListOptions{
				LabelSelector: k8s.NewLabels().
					ManagedByKrane().
					ToSelector(),
				Continue:  cursor,
				Namespace: "", // empty to match across all
			})

			if err != nil {
				c.logger.Error("unable to list sentinels", "error", err.Error())
				return
			}

			for _, sentinel := range sentinels.Items {

				sentinelID, ok := k8s.GetSentinelID(sentinel.GetLabels())

				if !ok {
					c.logger.Warn("skipping non-sentinel sentinel", "name", sentinel.Name)
					continue
				}
				sentinelIDs <- sentinelID
			}
			cursor = sentinels.Continue
			if cursor == "" {
				break
			}
		}

	}()
	return sentinelIDs
}
