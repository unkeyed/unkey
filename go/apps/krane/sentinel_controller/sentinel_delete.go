package sentinelcontroller

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteSentinel removes Sentinel CRDs with the specified sentinel ID.
//
// This method finds and deletes all Sentinel custom resources matching the
// given sentinel ID across all namespaces. The controller-runtime reconciler
// will handle the actual Kubernetes resource cleanup based on these CRD deletions.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: Sentinel deletion request containing the sentinel ID
//
// Returns an error if the listing operation fails or if any
// Sentinel CRD cannot be deleted.
func (c *SentinelController) DeleteSentinel(ctx context.Context, req *ctrlv1.DeleteSentinel) error {

	c.logger.Info("deleting sentinel",
		"sentinel_id", req.GetSentinelId(),
	)

	sentinelList := sentinelv1.SentinelList{} //nolint:exhaustruct
	if err := c.client.List(ctx, &sentinelList,
		&client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(
				k8s.NewLabels().
					ManagedByKrane().
					ToMap(),
			),

			Namespace: "", // empty to match across all
		},
	); err != nil {
		return fmt.Errorf("failed to list sentinels: %w", err)
	}

	if len(sentinelList.Items) == 0 {

		c.logger.Debug("sentinel had no CRD configured", "sentinel_id", req.GetSentinelId())
		return nil
	}

	for _, sentinel := range sentinelList.Items {
		err := c.client.Delete(ctx, &sentinel)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete sentinel resource %s: %w", sentinel.Name, err)
		}
	}

	c.logger.Info("sentinel deleted successfully",
		"sentinel_id", req.GetSentinelId(),
	)

	return nil
}
