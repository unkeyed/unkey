package reconciler

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Definitions to manage status conditions
const (
	// typeAvailableSentinel represents the status of the Deployment reconciliation
	typeAvailableSentinel = "Available"
)

var (
	requeueNow = ctrl.Result{Requeue: true, RequeueAfter: time.Millisecond}
	// requeuePeriodically is used to keep the cluster resoruces in sync with the database in case any change events go missing
	requeuePeriodically = ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Minute}
	requeueNever        = ctrl.Result{Requeue: false, RequeueAfter: 0}
)

// Reconcile handles reconciliation of Sentinel custom resources.
//
// This method implements the controller-runtime reconcile loop for Sentinel resources.
// It fetches the Sentinel resource, creates or updates the underlying
// Kubernetes StatefulSet, and manages resource status and conditions.
//
// The reconciliation process:
//  1. Fetch the Sentinel CRD
//  2. Create or update corresponding StatefulSet
//  3. Update status conditions based on operation results
//  4. Return appropriate requeue timing
//
// Parameters:
//   - ctx: Context for reconciliation operation
//   - req: Reconciliation request with namespace and name
//
// Returns ctrl.Result indicating when to requeue and any error encountered.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	sentinel := &apiv1.Sentinel{} // nolint:exhaustruct
	err := r.client.Get(ctx, req.NamespacedName, sentinel)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The sentinel resource is not found, it must be deleted
			return requeueNever, nil
		}
		// Error reading the object - requeue the request.
		r.logger.Error("failed to get sentinel", "error", err)
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
		if err = r.client.Status().Update(ctx, sentinel); err != nil {
			r.logger.Error("Failed to update Sentinel status", "error", err)
			return ctrl.Result{}, err
		}
		return requeueNow, nil
	}

	deployment, err := r.ensureDeploymentExists(ctx, sentinel)
	if err != nil {

		return ctrl.Result{}, err
	}

	svc, err := r.ensureServiceExists(ctx, sentinel)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, fn := range []func(ctx context.Context, sentinel *apiv1.Sentinel, deployment *appsv1.Deployment, service *corev1.Service) (bool, error){
		r.reconcileImage,
		r.reconcileReplicas,
		//r.reconcileCPU,
		//r.reconcileMemory,
	} {

		requeue, err := fn(ctx, sentinel, deployment, svc)
		if err != nil {
			return ctrl.Result{}, err
		}
		if requeue {
			r.logger.Info("Requeueing due to state change")
			return requeueNow, nil
		}
	}

	return requeuePeriodically, nil

}
