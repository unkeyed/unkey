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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/controller/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
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
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sentinel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
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
		meta.SetStatusCondition(&sentinel.Status.Conditions, metav1.Condition{Type: typeAvailableSentinel, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, sentinel); err != nil {
			log.Error(err, "Failed to update Sentinel status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the sentinel Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, sentinel); err != nil {
			log.Error(err, "Failed to re-fetch sentinel")
			return ctrl.Result{}, err
		}
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: sentinel.Name, Namespace: sentinel.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForSentinel(sentinel)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for Sentinel")

			// The following implementation will update the status
			meta.SetStatusCondition(&sentinel.Status.Conditions, metav1.Condition{Type: typeAvailableSentinel,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", sentinel.Name, err)})

			if err := r.Status().Update(ctx, sentinel); err != nil {
				log.Error(err, "Failed to update Sentinel status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// The CRD API defines that the Sentinel type have a SentinelSpec.Size field
	// to set the quantity of Deployment instances to the desired state on the cluster.
	// Therefore, the following code will ensure the Deployment size is the same as defined
	// via the Size spec of the Custom Resource which we are reconciling.
	if found.Spec.Replicas == nil || *found.Spec.Replicas != sentinel.Spec.Replicas {
		found.Spec.Replicas = ptr.To(sentinel.Spec.Replicas)
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			// Re-fetch the sentinel Custom Resource before updating the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, sentinel); err != nil {
				log.Error(err, "Failed to re-fetch sentinel")
				return ctrl.Result{}, err
			}

			// The following implementation will update the status
			meta.SetStatusCondition(&sentinel.Status.Conditions, metav1.Condition{Type: typeAvailableSentinel,
				Status: metav1.ConditionFalse, Reason: "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", sentinel.Name, err)})

			if err := r.Status().Update(ctx, sentinel); err != nil {
				log.Error(err, "Failed to update Sentinel status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		// Now, that we update the size we want to requeue the reconciliation
		// so that we can ensure that we have the latest state of the resource before
		// update. Also, it will help ensure the desired state on the cluster
		return ctrl.Result{Requeue: true}, nil
	}

	// The following implementation will update the status
	meta.SetStatusCondition(&sentinel.Status.Conditions, metav1.Condition{Type: typeAvailableSentinel,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", sentinel.Name, sentinel.Spec.Replicas)})

	if err := r.Status().Update(ctx, sentinel); err != nil {
		log.Error(err, "Failed to update Sentinel status")
		return ctrl.Result{}, err
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

// deploymentForSentinel returns a Sentinel Deployment object
func (r *SentinelReconciler) deploymentForSentinel(sentinel *apiv1.Sentinel) (*appsv1.Deployment, error) {

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sentinel.Name,
			Namespace: sentinel.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &sentinel.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "project"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "project"},
				},
				Spec: corev1.PodSpec{
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
							ContainerPort: 11211,
							Name:          "sentinel",
						}},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(sentinel, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}
