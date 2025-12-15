package deploymentcontroller

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	"github.com/unkeyed/unkey/go/pkg/ptr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// we requeue every non-deleted object to periodically reconcile against the desired state
var (
	requeueLater = ctrl.Result{RequeueAfter: 10 * time.Minute} // nolint:exhaustruct
	requeueNow   = ctrl.Result{RequeueAfter: time.Second}      // nolint:exhaustruct
	requeueNever = ctrl.Result{RequeueAfter: 0}                // nolint:exhaustruct
)

func (dc *DeploymentController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := dc.logger.With(
		"namespace", req.Namespace,
		"object_name", req.Name,
	)

	logger.Info("reconciling deployment")

	deploymentsClient := dc.clientset.AppsV1().Deployments(req.Namespace)

	var current v1.UnkeyDeployment
	err := dc.manager.GetClient().Get(ctx, req.NamespacedName, &current)
	if err != nil {
		if k8serrors.IsNotFound(err) { // not found, we can delete the resources
			logger.Info("deployment crd not found, deleting...")
			err = deploymentsClient.Delete(ctx, deploymentName(req.Name), metav1.DeleteOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					dc.logger.Debug("deployment was already deleted")
					return requeueNever, nil
				}
				return requeueLater, fmt.Errorf("couldn't delete deployment: %s", err)
			}

			return requeueNever, nil
		}
		return requeueNow, err
	}

	desiredDeployment := buildDeploymentObject(current)

	deployment, err := deploymentsClient.Get(ctx, desiredDeployment.GetName(), metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Define and create a new deployment.

			if _, err = deploymentsClient.Create(ctx, desiredDeployment, metav1.CreateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
			dc.logger.Info("new deployment created")
			return requeueNow, nil
		}
		return ctrl.Result{}, err
	}

	// Updates
	needsUpdate := false

	// Check replicas
	if *deployment.Spec.Replicas != current.Spec.Replicas {
		deployment.Spec.Replicas = ptr.P(current.Spec.Replicas)
		needsUpdate = true
	}

	// Check container image and resources
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		if container.Name != "container" {
			continue
		}

		// Check if image changed
		if container.Image != current.Spec.Image {
			// For significant changes like image or resources, update the entire pod template
			deployment.Spec.Template = desiredDeployment.Spec.Template
			needsUpdate = true
			break
		}

		// Check if resources changed
		desiredCPU := resource.NewMilliQuantity(current.Spec.CpuMillicores, resource.DecimalSI)
		desiredMem := resource.NewQuantity(current.Spec.MemoryMib*1024*1024, resource.DecimalSI)

		if !container.Resources.Requests.Cpu().Equal(*desiredCPU) ||
			!container.Resources.Requests.Memory().Equal(*desiredMem) {
			deployment.Spec.Template = desiredDeployment.Spec.Template
			needsUpdate = true
			break
		}
	}

	if needsUpdate {
		return dc.updateDeployment(ctx, deployment)
	}

	dc.logger.Info("deployment is fully synced")
	return requeueNever, nil
}

func deploymentName(crdName string) string {
	return fmt.Sprintf("%s-d", crdName)
}

func buildDeploymentObject(desired v1.UnkeyDeployment) *appsv1.Deployment {

	labels := k8s.NewLabels().
		WorkspaceID(desired.Spec.WorkspaceId).
		ProjectID(desired.Spec.ProjectId).
		EnvironmentID(desired.Spec.EnvironmentId).
		DeploymentID(desired.Spec.DeploymentId).
		ManagedByKrane().
		ToMap()

	// Use default Kubernetes rolling update strategy
	maxSurge := intstr.FromString("25%")
	maxUnavailable := intstr.FromString("25%")

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: desired.Namespace,
			Name:      deploymentName(desired.Name),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.P[int32](desired.Spec.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "container",
							Image: desired.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
							Env: []corev1.EnvVar{
								{Name: "UNKEY_PROJECT_ID", Value: desired.Spec.ProjectId},
								{Name: "UNKEY_ENVIRONMENT_ID", Value: desired.Spec.EnvironmentId},
								{Name: "UNKEY_DEPLOYMENT_ID", Value: desired.Spec.DeploymentId},
							},
							Resources: corev1.ResourceRequirements{
								// nolint: exhaustive
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(desired.Spec.CpuMillicores, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(desired.Spec.MemoryMib*1024*1024, resource.DecimalSI), //nolint: gosec

								},
								// nolint: exhaustive
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(desired.Spec.CpuMillicores, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(desired.Spec.MemoryMib*1024*1024, resource.DecimalSI), //nolint: gosec
								},
							},
						},
					},
				},
			},
		},
	}
}

func (dc *DeploymentController) updateDeployment(ctx context.Context, deployment *appsv1.Deployment) (ctrl.Result, error) {
	_, err := dc.clientset.AppsV1().Deployments(deployment.GetNamespace()).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	return requeueNow, nil
}
