package sentinelcontroller

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *SentinelController) GetRunningSentinelIDs(ctx context.Context) <-chan string {
	sentinelIDs := make(chan string)

	go func() {

		defer close(sentinelIDs)

		gws := sentinelv1.SentinelList{} // nolint:exhaustruct
		err := c.mgr.GetClient().List(ctx, &gws, &client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(
				k8s.NewLabels().
					ManagedByKrane().
					ToMap(),
			),

			Namespace: "", // empty to match across all
		})

		if err != nil {
			c.logger.Error("unable to list sentinels", "error", err.Error())
			return
		}

		for _, sentinel := range gws.Items {

			sentinelID, ok := k8s.GetSentinelID(sentinel.GetLabels())

			if !ok {
				c.logger.Warn("skipping non-sentinel sentinel", "name", sentinel.Name)
				continue
			}
			sentinelIDs <- sentinelID
		}

	}()
	return sentinelIDs
}
