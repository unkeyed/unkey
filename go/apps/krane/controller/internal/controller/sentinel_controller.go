/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

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

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Definitions to manage status conditions
const (
	// typeAvailableSentinel represents the status of the Deployment reconciliation
	typeAvailableSentinel = "Available"
)

// SentinelReconciler reconciles a Sentinel object
type SentinelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubernetes.unkey.com,resources=sentinels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubernetes.unkey.com,resources=sentinels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubernetes.unkey.com,resources=sentinels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *SentinelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Sentinel instance
	// The purpose is check if the Custom Resource for the Kind Sentinel
	// is applied on the cluster if not we return nil to stop the reconciliation
	sentinel := &apiv1.Sentinel{}
	err := r.Get(ctx, req.NamespacedName, sentinel)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("sentinel resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get sentinel")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(sentinel.Status.Conditions) == 0 {
		meta.SetStatusCondition(&sentinel.Status.Conditions, metav1.Condition{
			Type:    typeAvailableSentinel,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err = r.Status().Update(ctx, sentinel); err != nil {
			log.Error(err, "Failed to update Sentinel status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	deployment, err := r.ensureDeploymentExists(ctx, sentinel)
	if err != nil {

		return ctrl.Result{}, err
	}

	svc, err := r.ensureServiceExists(ctx, sentinel)
	if err != nil {

		return ctrl.Result{}, err
	}

	for _, fn := range []stateReconciler{
		r.reconcileImage,
		r.reconcileReplicas,
		r.reconcileCPU,
		r.reconcileMemory,
	} {

		requeue, err := fn(ctx, sentinel, deployment, svc)
		if err != nil {
			return ctrl.Result{}, err
		}
		if requeue {
			log.Info("Requeueing due to state change")
			return ctrl.Result{RequeueAfter: time.Millisecond}, nil
		}
	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *SentinelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Sentinel{}).
		Owns(&appsv1.Deployment{}).
		Named("sentinel").
		Complete(r)
}

func (r *SentinelReconciler) ensureDeploymentExists(ctx context.Context, sentinel *apiv1.Sentinel) (*appsv1.Deployment, error) {

	name := fmt.Sprintf("%s-dpl", sentinel.Name)

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Namespace: sentinel.Namespace, Name: name}, found)

	if err == nil {
		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sentinel.Namespace,
			Labels: k8s.NewLabels().
				WorkspaceID(sentinel.Spec.WorkspaceID).
				ProjectID(sentinel.Spec.ProjectID).
				EnvironmentID(sentinel.Spec.EnvironmentID).
				SentinelID(sentinel.Spec.SentinelID).
				ManagedByKrane().
				ToMap(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &sentinel.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: k8s.NewLabels().SentinelID(sentinel.Spec.SentinelID).ToMap(),
			},
			Template: corev1.PodTemplateSpec{

				ObjectMeta: metav1.ObjectMeta{

					Labels: k8s.NewLabels().SentinelID(sentinel.Spec.SentinelID).ToMap(),
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
						Image:           sentinel.Spec.Image,
						Name:            "sentinel",
						ImagePullPolicy: corev1.PullIfNotPresent,

						Ports: []corev1.ContainerPort{{
							ContainerPort: 8040,
							Name:          "sentinel",
						}},

						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(sentinel.Spec.CpuMillicores, resource.BinarySI),
								corev1.ResourceMemory: *resource.NewQuantity(sentinel.Spec.MemoryMib, resource.BinarySI),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(sentinel.Spec.CpuMillicores, resource.BinarySI),
								corev1.ResourceMemory: *resource.NewQuantity(sentinel.Spec.MemoryMib, resource.BinarySI),
							},
						},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(sentinel, deployment, r.Scheme); err != nil {
		return nil, err
	}

	if err = r.Create(ctx, deployment); err != nil {

		return nil, err
	}

	return deployment, nil

}

func (r *SentinelReconciler) ensureServiceExists(ctx context.Context, sentinel *apiv1.Sentinel) (*corev1.Service, error) {

	name := fmt.Sprintf("%s-svc", sentinel.Name)

	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Namespace: sentinel.Namespace, Name: name}, found)

	if err == nil {
		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-svc", sentinel.Name),
			Namespace: sentinel.Namespace,
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
	if err := ctrl.SetControllerReference(sentinel, svc, r.Scheme); err != nil {
		return nil, err
	}

	if err = r.Create(ctx, svc); err != nil {

		return nil, err

	}

	return svc, nil

}

// for every spec field in the CRD we write a state reconciler
// returns true if the resource was changed and therefore should be requeued
type stateReconciler func(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, service *corev1.Service) (bool, error)

func (r *SentinelReconciler) reconcileImage(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling image")
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Image != sentinel.Spec.Image {
			deployment.Spec.Template.Spec.Containers[i].Image = sentinel.Spec.Image

			err := r.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}

	return false, nil

}

func (r *SentinelReconciler) reconcileReplicas(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	log := logf.FromContext(ctx)
	log.Info("Reconciling replicas")
	if *deployment.Spec.Replicas != sentinel.Spec.Replicas {
		deployment.Spec.Replicas = &sentinel.Spec.Replicas

		err := r.Update(ctx, deployment)
		if err != nil {
			return false, err
		}
		return true, nil

	}

	return false, nil

}

func (r *SentinelReconciler) reconcileCPU(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	log := logf.FromContext(ctx)
	log.Info("Reconciling CPU")
	for i := range deployment.Spec.Template.Spec.Containers {
		changes := false

		limit := deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Cpu()
		if limit.ScaledValue(resource.Milli) != sentinel.Spec.CpuMillicores {
			log.Info("Updating CPU limit", "container", deployment.Spec.Template.Spec.Containers[i].Name, "limit", limit.Value(), "desired", sentinel.Spec.CpuMillicores)
			deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Cpu().SetScaled(sentinel.Spec.CpuMillicores, resource.Milli)
			changes = true

		}
		request := deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Cpu()
		if request.ScaledValue(resource.Milli) != sentinel.Spec.CpuMillicores {
			log.Info("Updating CPU request", "container", deployment.Spec.Template.Spec.Containers[i].Name, "request", request.Value(), "desired", sentinel.Spec.CpuMillicores)

			deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Cpu().SetScaled(sentinel.Spec.CpuMillicores, resource.Milli)
			changes = true

		}
		if changes {
			log.Info("Updating container", "container", deployment.Spec.Template.Spec.Containers[i].Name, "index", i)

			err := r.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}
	return false, nil

}

func (r *SentinelReconciler) reconcileMemory(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	log := logf.FromContext(ctx)
	log.Info("Reconciling Memory")
	for i := range deployment.Spec.Template.Spec.Containers {
		changes := false

		limit := deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Memory()
		if limit.Value() != sentinel.Spec.MemoryMib {
			log.Info("Updating Memory limit", "container", deployment.Spec.Template.Spec.Containers[i].Name, "limit", limit.Value(), "desired", sentinel.Spec.MemoryMib)
			deployment.Spec.Template.Spec.Containers[i].Resources.Limits.Memory().Set(sentinel.Spec.MemoryMib)
			changes = true

		}
		request := deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Memory()
		if request.Value() != sentinel.Spec.MemoryMib {
			log.Info("Updating Memory request", "container", deployment.Spec.Template.Spec.Containers[i].Name, "request", request.Value(), "desired", sentinel.Spec.MemoryMib)

			deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Memory().Set(sentinel.Spec.MemoryMib)
			changes = true

		}
		if changes {
			log.Info("Updating container", "container", deployment.Spec.Template.Spec.Containers[i].Name, "index", i)

			err := r.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}

	return false, nil

}
