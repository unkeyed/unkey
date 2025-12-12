package sentinelcontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/pkg/ptr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// we requeue every non-deleted object to periodically reconcile against the desired state in planetscale
var (
	requeueLater = ctrl.Result{RequeueAfter: 10 * time.Minute} // nolint:exhaustruct
	requeueNow   = ctrl.Result{RequeueAfter: time.Second}      // nolint:exhaustruct
	requeueNever = ctrl.Result{RequeueAfter: 0}                // nolint:exhaustruct
)

func (gc *SentinelController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := gc.logger.With(
		"namespace", req.Namespace,
		"object_name", req.Name,
	)

	logger.Info("reconciling sentinel")

	deploymentsClient := gc.clientset.AppsV1().Deployments(req.Namespace)
	serviceClient := gc.clientset.CoreV1().Services(req.Namespace)

	var current sentinelv1.Sentinel
	err := gc.mgr.GetClient().Get(ctx, req.NamespacedName, &current)
	if err != nil {
		if k8serrors.IsNotFound(err) { // not found, we can delete the resources
			logger.Info("sentinel crd not found, deleting...")
			err = deploymentsClient.Delete(ctx, deploymentName(req.Name), metav1.DeleteOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					gc.logger.Debug("deployment was already deleted")
					return requeueNever, nil
				}
				return requeueLater, fmt.Errorf("couldn't delete deployment: %s", err)
			}

			err = serviceClient.Delete(ctx, serviceName(req.Name), metav1.DeleteOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					gc.logger.Debug("service was already deleted")
					return requeueNever, nil
				}
				return requeueLater, fmt.Errorf("couldn't delete service: %s", err)
			}

			return requeueNever, nil
		}
		return requeueNow, err
	}

	desiredDeployment, desiredService := buildDeploymentObject(current)

	deployment, err := deploymentsClient.Get(ctx, desiredDeployment.GetName(), metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Define and create a new deployment.

			if _, err = deploymentsClient.Create(ctx, desiredDeployment, metav1.CreateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
			gc.logger.Info("new sentinel deployment created")
			return requeueNow, nil
		}
		return ctrl.Result{}, err
	}

	service, err := serviceClient.Get(ctx, desiredService.GetName(), metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Define and create a new deployment.
			gc.logger.Warn("service not found", "getName", desiredService.GetName())

			if _, err = serviceClient.Create(ctx, desiredService, metav1.CreateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
			gc.logger.Info("new sentinel service created")
			return requeueNow, nil
		}
		return ctrl.Result{}, err
	}

	if service.OwnerReferences == nil {
		service.OwnerReferences = []metav1.OwnerReference{}
	}

	hasCorrectOwnerReference := false
	for _, ref := range service.OwnerReferences {
		if ref.UID == deployment.UID {
			hasCorrectOwnerReference = true
			break
		}
	}
	if !hasCorrectOwnerReference {

		service.OwnerReferences = append(service.OwnerReferences, metav1.OwnerReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       deployment.Name,
			UID:        deployment.UID,
		})

		return gc.updateService(ctx, service)
	}

	// Updates

	for i := range deployment.Spec.Template.Spec.Containers {
		if deployment.Spec.Template.Spec.Containers[i].Image != current.Spec.Image {
			deployment.Spec.Template.Spec.Containers[i].Image = current.Spec.Image
			return gc.updateDeployment(ctx, deployment)
		}
	}

	if *deployment.Spec.Replicas != current.Spec.Replicas {
		deployment.Spec.Replicas = ptr.P(current.Spec.Replicas)
		return gc.updateDeployment(ctx, deployment)
	}

	gc.logger.Info("sentinel is fully synced")
	return requeueNever, nil
}

func deploymentName(crdName string) string {
	return fmt.Sprintf("%s-d", crdName)
}

func serviceName(crdName string) string {
	return fmt.Sprintf("%s-s", crdName)
}

func buildDeploymentObject(desired sentinelv1.Sentinel) (*appsv1.Deployment, *corev1.Service) {

	labels := k8s.NewLabels().WorkspaceID(desired.Spec.WorkspaceId).ProjectID(desired.Spec.ProjectId).EnvironmentID(desired.Spec.EnvironmentId).SentinelID(desired.Spec.SentinelId).ManagedByKrane().ToMap()

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
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "sentinel",
								Image:   desired.Spec.Image,
								Command: []string{"run", "sentinel"},
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 80,
									},
								},
								Env: []corev1.EnvVar{
									{Name: "UNKEY_WORKSPACE_ID", Value: desired.Spec.WorkspaceId},
									{Name: "UNKEY_PROJECT_ID", Value: desired.Spec.ProjectId},
									{Name: "UNKEY_GATEWAY_ID", Value: desired.Spec.SentinelId},
									{Name: "UNKEY_ENVIRONMENT_ID", Value: desired.Spec.EnvironmentId},
									{Name: "UNKEY_IMAGE", Value: desired.Spec.Image},
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
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName(desired.Name),
				Namespace: desired.Namespace,
				Labels:    labels,
			},
			//nolint:exhaustruct
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
				Selector: labels,
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
}

func (gc *SentinelController) updateDeployment(ctx context.Context, deployment *appsv1.Deployment) (ctrl.Result, error) {
	_, err := gc.clientset.AppsV1().Deployments(deployment.GetNamespace()).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	return requeueNow, nil
}

func (gc *SentinelController) updateService(ctx context.Context, service *corev1.Service) (ctrl.Result, error) {
	_, err := gc.clientset.CoreV1().Services(service.GetNamespace()).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	return requeueNow, nil
}
