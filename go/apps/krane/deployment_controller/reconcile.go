package deploymentcontroller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Definitions to manage status conditions
const (
	// typeAvailableDeployment represents the status of the Deployment reconciliation
	typeAvailableDeployment = "Available"
)

// Reconcile handles reconciliation of Deployment custom resources.
//
// This method implements the controller-runtime reconcile loop for Deployment resources.
// It fetches the Deployment resource, creates or updates the underlying
// Kubernetes StatefulSet, and manages resource status and conditions.
//
// The reconciliation process:
//  1. Fetch the Deployment CRD
//  2. Create or update corresponding StatefulSet
//  3. Update status conditions based on operation results
//  4. Return appropriate requeue timing
//
// Parameters:
//   - ctx: Context for reconciliation operation
//   - req: Reconciliation request with namespace and name
//
// Returns ctrl.Result indicating when to requeue and any error encountered.
func (r *DeploymentController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	target := &apiv1.Deployment{}
	err := r.client.Get(ctx, req.NamespacedName, target)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The deployment resource is not found, it must be deleted

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.logger.Error("failed to get deployment", "error", err)
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(target.Status.Conditions) == 0 {
		meta.SetStatusCondition(&target.Status.Conditions, metav1.Condition{
			Type:    typeAvailableDeployment,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err = r.client.Status().Update(ctx, target); err != nil {
			r.logger.Error("Failed to update Deployment status", "error", err)
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	deployment, err := r.ensureDeploymentExists(ctx, target)
	if err != nil {
		return ctrl.Result{}, err
	}

	svc, err := r.ensureServiceExists(ctx, target)
	if err != nil {

		return ctrl.Result{}, err
	}

	for _, fn := range []stateReconciler{
		r.reconcileImage,
		r.reconcileReplicas,
		r.reconcileCPU,
		r.reconcileMemory,
	} {

		requeue, err := fn(ctx, target, deployment, svc)
		if err != nil {
			return ctrl.Result{}, err

		}
		if requeue {
			r.logger.Info("Requeueing due to state change")
			return ctrl.Result{RequeueAfter: time.Millisecond}, nil
		}
	}

	return ctrl.Result{}, nil

}
func (r *DeploymentController) ensureDeploymentExists(ctx context.Context, target *apiv1.Deployment) (*appsv1.Deployment, error) {

	name := fmt.Sprintf("%s-dpl", target.Name)

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: target.Namespace, Name: name}, found)

	if err == nil {

		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: target.Namespace,
			Labels: k8s.NewLabels().
				WorkspaceID(target.Spec.WorkspaceID).
				ProjectID(target.Spec.ProjectID).
				EnvironmentID(target.Spec.EnvironmentID).
				DeploymentID(target.Spec.DeploymentID).
				ComponentDeployment().
				ManagedByKrane().
				ToMap(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &target.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: k8s.NewLabels().DeploymentID(target.Spec.DeploymentID).ToMap(),
			},
			Template: corev1.PodTemplateSpec{

				ObjectMeta: metav1.ObjectMeta{

					Labels: k8s.NewLabels().DeploymentID(target.Spec.DeploymentID).ToMap(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           target.Spec.Image,
						Name:            "deployment",
						ImagePullPolicy: corev1.PullIfNotPresent,

						Ports: []corev1.ContainerPort{{
							ContainerPort: 8040,
							Name:          "deployment",
						}},

						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(target.Spec.CpuMillicores, resource.BinarySI),
								corev1.ResourceMemory: *resource.NewQuantity(target.Spec.MemoryMib, resource.BinarySI),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(target.Spec.CpuMillicores, resource.BinarySI),
								corev1.ResourceMemory: *resource.NewQuantity(target.Spec.MemoryMib, resource.BinarySI),
							},
						},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(target, deployment, r.scheme); err != nil {
		return nil, err
	}

	if err = r.client.Create(ctx, deployment); err != nil {

		return nil, err
	}

	return deployment, nil

}

func (r *DeploymentController) ensureServiceExists(ctx context.Context, deployment *apiv1.Deployment) (*corev1.Service, error) {

	name := fmt.Sprintf("%s-svc", deployment.Name)

	found := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: deployment.Namespace, Name: name}, found)

	if err == nil {

		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-svc", deployment.Name),
			Namespace: deployment.Namespace,
			Labels: k8s.NewLabels().
				WorkspaceID(deployment.Spec.WorkspaceID).
				ProjectID(deployment.Spec.ProjectID).
				EnvironmentID(deployment.Spec.EnvironmentID).
				DeploymentID(deployment.Spec.DeploymentID).
				ManagedByKrane().
				ToMap(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
			Selector: k8s.NewLabels().DeploymentID(deployment.Spec.DeploymentID).ToMap(),
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
	if err := ctrl.SetControllerReference(deployment, svc, r.scheme); err != nil {
		return nil, err
	}

	if err = r.client.Create(ctx, svc); err != nil {

		return nil, err

	}

	return svc, nil

}

// for every spec field in the CRD we write a state reconciler
// returns true if the resource was changed and therefore should be requeued
type stateReconciler func(ctx context.Context, target *apiv1.Deployment, deployment *appsv1.Deployment, service *corev1.Service) (bool, error)

func (r *DeploymentController) reconcileImage(ctx context.Context, target *apiv1.Deployment, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {
	r.logger.Info("Reconciling image")
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Image != target.Spec.Image {
			deployment.Spec.Template.Spec.Containers[i].Image = target.Spec.Image

			err := r.client.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}

	return false, nil

}

func (r *DeploymentController) reconcileReplicas(ctx context.Context, target *apiv1.Deployment, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	r.logger.Info("Reconciling replicas")
	if *deployment.Spec.Replicas != target.Spec.Replicas {
		deployment.Spec.Replicas = &target.Spec.Replicas

		err := r.client.Update(ctx, deployment)
		if err != nil {
			return false, err
		}
		return true, nil

	}

	return false, nil

}

func (r *DeploymentController) reconcileCPU(ctx context.Context, target *apiv1.Deployment, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	r.logger.Info("Reconciling CPU")
	for i := range deployment.Spec.Template.Spec.Containers {
		changes := false

		limit := deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Cpu()
		if limit.ScaledValue(resource.Milli) != target.Spec.CpuMillicores {
			r.logger.Info("Updating CPU limit", "container", deployment.Spec.Template.Spec.Containers[i].Name, "limit", limit.Value(), "desired", target.Spec.CpuMillicores)
			deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Cpu().SetScaled(target.Spec.CpuMillicores, resource.Milli)
			changes = true

		}
		request := deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Cpu()
		if request.ScaledValue(resource.Milli) != target.Spec.CpuMillicores {
			r.logger.Info("Updating CPU request", "container", deployment.Spec.Template.Spec.Containers[i].Name, "request", request.Value(), "desired", target.Spec.CpuMillicores)

			deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Cpu().SetScaled(target.Spec.CpuMillicores, resource.Milli)
			changes = true

		}
		if changes {
			r.logger.Info("Updating container", "container", deployment.Spec.Template.Spec.Containers[i].Name, "index", i)

			err := r.client.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}
	return false, nil

}

func (r *DeploymentController) reconcileMemory(ctx context.Context, target *apiv1.Deployment, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	r.logger.Info("Reconciling Memory")
	for i := range deployment.Spec.Template.Spec.Containers {
		changes := false

		limit := deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Memory()
		if limit.Value() != target.Spec.MemoryMib {
			r.logger.Info("Updating Memory limit", "container", deployment.Spec.Template.Spec.Containers[i].Name, "limit", limit.Value(), "desired", target.Spec.MemoryMib)
			deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Memory().Set(target.Spec.MemoryMib)
			changes = true

		}
		request := deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Memory()
		if request.Value() != target.Spec.MemoryMib {
			r.logger.Info("Updating Memory request", "container", deployment.Spec.Template.Spec.Containers[i].Name, "request", request.Value(), "desired", target.Spec.MemoryMib)

			deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Memory().Set(target.Spec.MemoryMib)
			changes = true

		}
		if changes {
			r.logger.Info("Updating container", "container", deployment.Spec.Template.Spec.Containers[i].Name, "index", i)

			err := r.client.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}

	return false, nil

}

func k8sPodStatusToUnkeyPodStatus(status corev1.PodStatus) ctrlv1.UpdateInstanceRequest_Status {
	switch status.Phase {
	case corev1.PodPending:
		return ctrlv1.UpdateInstanceRequest_STATUS_PENDING
	case corev1.PodRunning:
		return ctrlv1.UpdateInstanceRequest_STATUS_RUNNING
	case corev1.PodSucceeded:
		return ctrlv1.UpdateInstanceRequest_STATUS_UNSPECIFIED
	case corev1.PodFailed:
		return ctrlv1.UpdateInstanceRequest_STATUS_FAILED
	case corev1.PodUnknown:
		return ctrlv1.UpdateInstanceRequest_STATUS_UNSPECIFIED
	}

	return ctrlv1.UpdateInstanceRequest_STATUS_UNSPECIFIED
}
