package reconciler

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Definitions to manage status conditions
const (
	// typeAvailableSentinel represents the status of the Deployment reconciliation.
	// This condition indicates whether the sentinel deployment is fully operational
	// and ready to serve traffic.
	typeAvailableSentinel = "Available"
)

var (
	// requeue requests immediate requeue with minimal delay.
	// Used when state changes require immediate attention.
	requeue = ctrl.Result{Requeue: true, RequeueAfter: time.Millisecond}
	// forget indicates no further reconciliation is needed.
	// Used for successful completions and when resources are deleted.
	forget = ctrl.Result{Requeue: false, RequeueAfter: 0}
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
			return forget, nil
		}
		// Error reading the object - requeue the request.
		r.logger.Error("failed to get sentinel", "error", err)
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(sentinel.Status.Conditions) == 0 {
		return r.setStatusAndReqeue(ctx, sentinel, metav1.ConditionUnknown, "Starting reconciliation")
	}

	deployment, err := r.ensureDeploymentExists(ctx, sentinel)
	if err != nil {
		return r.setStatusAndReqeue(ctx, sentinel, metav1.ConditionFalse, "deployment could not be created")
	}

	svc, err := r.ensureServiceExists(ctx, sentinel)
	if err != nil {
		return r.setStatusAndReqeue(ctx, sentinel, metav1.ConditionFalse, "service could not be created")
	}

	var _ = svc

	status := metav1.ConditionFalse
	if deployment.Status.AvailableReplicas >= sentinel.Spec.Replicas {
		status = metav1.ConditionTrue
	}
	return forget, r.setStatus(ctx, sentinel, status, "reconcile finished")

}

// setStatus updates the sentinel's status condition.
//
// This helper function sets the Available condition on the sentinel resource
// with the provided status and message. It uses the standard Kubernetes
// condition pattern with "Reconciling" as the reason.
func (r *Reconciler) setStatus(ctx context.Context, sentinel *apiv1.Sentinel, status metav1.ConditionStatus, message string) error {
	meta.SetStatusCondition(&sentinel.Status.Conditions, metav1.Condition{
		Type:    typeAvailableSentinel,
		Status:  status,
		Reason:  "Reconciling",
		Message: message,
	})
	return r.client.Status().Update(ctx, sentinel)
}

// setStatusAndReqeue updates status and requests immediate requeue.
//
// This helper function combines status updates with immediate requeuing,
// which is commonly needed when transitioning between reconciliation states.
// The requeue ensures the controller continues processing the updated state.
func (r *Reconciler) setStatusAndReqeue(ctx context.Context, sentinel *apiv1.Sentinel, status metav1.ConditionStatus, message string) (ctrl.Result, error) {
	err := r.setStatus(ctx, sentinel, status, message)
	if err != nil {
		return ctrl.Result{}, err
	}
	return requeue, nil
}
