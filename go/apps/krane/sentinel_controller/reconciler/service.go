package reconciler

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) ensureServiceExists(ctx context.Context, sentinel *apiv1.Sentinel) (*corev1.Service, error) {

	name := fmt.Sprintf("%s-svc", sentinel.GetName())

	found := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: sentinel.GetNamespace(), Name: name}, found)

	if err == nil {

		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sentinel.GetNamespace(),
			Labels: k8s.NewLabels().
				WorkspaceID(sentinel.Spec.WorkspaceID).
				ProjectID(sentinel.Spec.ProjectID).
				EnvironmentID(sentinel.Spec.EnvironmentID).
				SentinelID(sentinel.Spec.SentinelID).
				ManagedByKrane().
				ToMap(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
			Selector: k8s.NewLabels().SentinelID(sentinel.Spec.SentinelID).ToMap(),
			//nolint:exhaustruct
			Ports: []corev1.ServicePort{
				{
					Port:       8040,
					TargetPort: intstr.FromInt(8040),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(sentinel, svc, r.scheme); err != nil {
		return nil, err
	}

	if err = r.client.Create(ctx, svc); err != nil {
		return nil, err
	}

	return svc, nil

}
