package reconciler

import (
	"context"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (r *Reconciler) reconcileImage(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {
	r.logger.Info("Reconciling image")
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Image != sentinel.Spec.Image {
			deployment.Spec.Template.Spec.Containers[i].Image = sentinel.Spec.Image

			err := r.client.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}

	return false, nil

}
