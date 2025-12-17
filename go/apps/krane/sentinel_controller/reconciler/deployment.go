package reconciler

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) ensureDeploymentExists(ctx context.Context, sentinel *apiv1.Sentinel) (*appsv1.Deployment, error) {

	name := fmt.Sprintf("%s-dpl", sentinel.GetName())
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sentinel.GetNamespace(),
			Labels: k8s.NewLabels().
				WorkspaceID(sentinel.Spec.WorkspaceID).
				ProjectID(sentinel.Spec.ProjectID).
				EnvironmentID(sentinel.Spec.EnvironmentID).
				SentinelID(sentinel.Spec.SentinelID).
				ComponentSentinel().
				ManagedByKrane().
				ToMap(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.P(sentinel.Spec.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: k8s.NewLabels().SentinelID(sentinel.Spec.SentinelID).ToMap(),
			},
			Template: corev1.PodTemplateSpec{

				ObjectMeta: metav1.ObjectMeta{

					Labels: k8s.NewLabels().SentinelID(sentinel.Spec.SentinelID).ToMap(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,

					Containers: []corev1.Container{{
						Image:           sentinel.Spec.Image,
						Name:            "sentinel",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Args:            []string{"run", "sentinel"},
						Env: []corev1.EnvVar{
							{Name: "UNKEY_WORKSPACE_ID", Value: sentinel.Spec.WorkspaceID},
							{Name: "UNKEY_PROJECT_ID", Value: sentinel.Spec.ProjectID},
							{Name: "UNKEY_ENVIRONMENT_ID", Value: sentinel.Spec.EnvironmentID},
							{Name: "UNKEY_SENTINEL_ID", Value: sentinel.Spec.SentinelID},
							{Name: "UNKEY_DATABASE_PRIMARY", Value: "unkey:password@tcp(mysql:3306)/unkey?parseTime=true&interpolateParams=true"},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: 8040,
							Name:          "sentinel",
						}},

						//Resources: corev1.ResourceRequirements{
						//	// nolint:exhaustive
						//	Limits: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(sentinel.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(sentinel.GetMemoryMib()), resource.BinarySI),
						//	},
						//	// nolint:exhaustive
						//	Requests: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(sentinel.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(sentinel.GetMemoryMib()), resource.BinarySI),
						//	},
						//},
					}},
				},
			},
		},
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: sentinel.GetNamespace(), Name: name}, found)

	if err == nil {

		found.Spec = deployment.Spec
		if err := r.client.Update(ctx, found); err != nil {
			return nil, err
		}

		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(sentinel, deployment, r.scheme); err != nil {
		return nil, err
	}

	if err = r.client.Create(ctx, deployment); err != nil {

		return nil, err
	}

	return deployment, nil

}
