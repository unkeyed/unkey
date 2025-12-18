package reflector

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

// deleteSentinel removes a Sentinel custom resource from the Kubernetes cluster.
//
// This function handles the deletion of sentinel resources when the control
// plane indicates they should be removed. The function gracefully handles
// cases where the sentinel resource doesn't already exist.
//
// Parameters:
//   - ctx: Context for the delete operation
//   - req: Delete request containing namespace and name of sentinel to delete
//
// Returns an error if the delete operation fails (excluding not found errors,
// which are ignored gracefully). The deletion cascades to owned resources
// (Deployments, Services) through Kubernetes garbage collection.
func (r *Reflector) deleteSentinel(ctx context.Context, req *ctrlv1.DeleteSentinel) error {
	r.logger.Info("deleting sentinel",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
	)

	// nolint:exhaustruct
	err := r.client.Delete(ctx, &sentinelv1.Sentinel{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.GetK8SNamespace(),
			Name:      req.GetK8SName(),
		},
	})

	return client.IgnoreNotFound(err)
}
