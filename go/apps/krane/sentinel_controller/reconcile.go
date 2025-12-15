package sentinelcontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
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

func (c *SentinelController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := c.logger.With(
		"namespace", req.Namespace,
		"object_name", req.Name,
	)

	logger.Info("reconciling sentinel")

	// Try to get the config to see what API server it's connecting to
	logger.Info("DEBUG: Attempting to get deployment to see connection details")

	current := &sentinelv1.Sentinel{}
	err := c.manager.GetClient().Get(ctx, req.NamespacedName, current)
	if err != nil {
		if k8serrors.IsNotFound(err) { // not found, we can delete the resources
			logger.Info("sentinel crd not found, deleting...")

			err = c.manager.GetClient().Delete(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: req.Namespace,
					Name:      serviceName(req.Name),
				},
			})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					c.logger.Debug("service was already deleted")
					return requeueNever, nil
				}
				return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
			}

			logger.Info("deleting sentinel deployment")
			err = c.client.AppsV1().Deployments(req.Namespace).Delete(ctx, deploymentName(req.Name), metav1.DeleteOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					c.logger.Debug("deployment was already deleted")
					return requeueNever, nil
				}
				return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
			}

			return requeueNever, nil
		}
		logger.Error("sentinel.get error", "error", err.Error())
		return ctrl.Result{}, err
	}

	logger.Info("sentinel existed")

	desiredDeployment, desiredService := buildDeploymentObject(current)

	deployment, err := c.client.AppsV1().Deployments(req.Namespace).Get(ctx, deploymentName(req.Name), metav1.GetOptions{})
	logger.Info("get deployment", "deployment", deployment, "error", err.Error())

	if err != nil {
		if k8serrors.IsNotFound(err) {
			if _, err = c.client.AppsV1().Deployments(req.Namespace).Create(ctx, desiredDeployment, metav1.CreateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
			c.logger.Info("new sentinel deployment created")
			return requeueNow, nil
		}
		return ctrl.Result{}, err
	}

	service, err := c.client.CoreV1().Services(req.Namespace).Get(ctx, desiredService.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Define and create a new deployment.
			c.logger.Warn("service not found", "getName", desiredService.GetName())

			if _, err = c.client.CoreV1().Services(req.Namespace).Create(ctx, desiredService, metav1.CreateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
			c.logger.Info("new sentinel service created")
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

		return c.updateService(ctx, service)
	}

	// Updates

	for i := range deployment.Spec.Template.Spec.Containers {
		if deployment.Spec.Template.Spec.Containers[i].Image != current.Spec.Image {
			deployment.Spec.Template.Spec.Containers[i].Image = current.Spec.Image
			return c.updateDeployment(ctx, deployment)
		}
	}

	if *deployment.Spec.Replicas != current.Spec.Replicas {
		deployment.Spec.Replicas = ptr.P(current.Spec.Replicas)
		return c.updateDeployment(ctx, deployment)
	}

	c.logger.Info("sentinel is fully synced")
	return requeueNever, nil
}

func deploymentName(crdName string) string {
	return fmt.Sprintf("%s-d", crdName)
}

func serviceName(crdName string) string {
	return fmt.Sprintf("%s-s", crdName)
}

func buildDeploymentObject(desired *sentinelv1.Sentinel) (*appsv1.Deployment, *corev1.Service) {

	labels := k8s.NewLabels().WorkspaceID(desired.Spec.WorkspaceID).ProjectID(desired.Spec.ProjectID).EnvironmentID(desired.Spec.EnvironmentID).SentinelID(desired.Spec.SentinelID).ManagedByKrane().ToMap()

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
									{Name: "UNKEY_WORKSPACE_ID", Value: desired.Spec.WorkspaceID},
									{Name: "UNKEY_PROJECT_ID", Value: desired.Spec.ProjectID},
									{Name: "UNKEY_SENTINEL_ID", Value: desired.Spec.SentinelID},
									{Name: "UNKEY_ENVIRONMENT_ID", Value: desired.Spec.EnvironmentID},
									{Name: "UNKEY_IMAGE", Value: desired.Spec.Image},
								},
								Resources: corev1.ResourceRequirements{
									// nolint: exhaustive
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewScaledQuantity(desired.Spec.CpuMillicores, resource.Milli),
										corev1.ResourceMemory: *resource.NewScaledQuantity(desired.Spec.MemoryMib, resource.Mega), //nolint: gosec

									},
									// nolint: exhaustive
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    *resource.NewScaledQuantity(desired.Spec.CpuMillicores, resource.Milli),
										corev1.ResourceMemory: *resource.NewScaledQuantity(desired.Spec.MemoryMib, resource.Mega), //nolint: gosec
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

func (c *SentinelController) updateDeployment(ctx context.Context, deployment *appsv1.Deployment) (ctrl.Result, error) {

	_, err := c.client.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	return requeueNow, nil
}

func (c *SentinelController) updateService(ctx context.Context, service *corev1.Service) (ctrl.Result, error) {

	_, err := c.client.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	return requeueNow, nil
}
