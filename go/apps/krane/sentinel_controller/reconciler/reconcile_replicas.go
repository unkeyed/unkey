package reconciler

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
)

func (r *Reconciler) reconcileReplicas(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	r.logger.Info("Reconciling replicas")
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != sentinel.Spec.Replicas {
		deployment.Spec.Replicas = ptr.P(sentinel.Spec.Replicas)

		err := r.client.Update(ctx, deployment)
		if err != nil {
			return false, err
		}
		return true, nil

	}

	return false, nil

}
