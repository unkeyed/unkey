package reconciler

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

// DeleteSentinel removes a sentinel's Service and Deployment from the cluster.
//
// Both resources are deleted explicitly rather than relying on owner reference
// cascading, ensuring cleanup completes even if ownership wasn't set correctly.
// Not-found errors are ignored since the desired end state is already achieved.
func (r *Reconciler) DeleteSentinel(ctx context.Context, req *ctrlv1.DeleteSentinel) error {
	r.logger.Info("deleting sentinel",
		"namespace", NamespaceSentinel,
		"name", req.GetK8SName(),
	)

	err := r.clientSet.CoreV1().Services(NamespaceSentinel).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	err = r.clientSet.AppsV1().Deployments(NamespaceSentinel).Delete(ctx, req.GetK8SName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	err = r.updateSentinelState(ctx, &ctrlv1.UpdateSentinelStateRequest{
		K8SName:           req.GetK8SName(),
		AvailableReplicas: 0,
	})
	if err != nil {
		return err
	}

	return client.IgnoreNotFound(err)
}
