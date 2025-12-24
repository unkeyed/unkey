package deployment

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deleteDeployment removes a Deployment custom resource from the Kubernetes cluster.
//
// This function handles the deletion of deployment resources when the control
// plane indicates they should be removed. The function gracefully handles
// cases where the deployment resource doesn't already exist.
//
// Parameters:
//   - ctx: Context for the delete operation
//   - req: Delete request containing namespace and name of deployment to delete
//
// Returns an error if the delete operation fails (excluding not found errors,
// which are ignored gracefully). The deletion cascades to owned resources
// (Deployments, Services) through Kubernetes garbage collection.
func (r *Reconciler) deleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) error {
	r.logger.Info("deleting deployment",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
	)

	// nolint:exhaustruct
	err := r.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace()).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	err = r.updateState(ctx, &ctrlv1.UpdateDeploymentStateRequest{
		Change: &ctrlv1.UpdateDeploymentStateRequest_Delete_{
			Delete: &ctrlv1.UpdateDeploymentStateRequest_Delete{
				K8SName: req.GetK8SName(),
			},
		},
	})
	if err != nil {
		return err
	}

	return client.IgnoreNotFound(err)
}
