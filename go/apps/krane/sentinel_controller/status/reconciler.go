// Package status implements status reporting for Sentinel deployments.
//
// This package contains the status reconciler that monitors Deployment resources
// for sentinel components and reports their operational status back to the control
// plane. This creates a bidirectional sync between Kubernetes cluster state
// and the centralized control plane database.
//
// # Status Reporting Flow
//
// The status reconciler follows a monitor-and-report pattern:
//  1. Watches sentinel Deployment resources for status changes
//  2. Extracts operational metrics (available replicas)
//  3. Reports metrics to control plane via gRPC
//  4. Uses circuit breaker to handle control plane unavailability
//  5. Periodically rechecks status for consistency
//
// # Key Components
//
//   - [StatusReconciler]: Main reconciler implementing controller-runtime.Reconciler
//   - Circuit breaker integration for resilient control plane communication
//   - Predicate-based filtering to process only sentinel deployments
//
// # Integration Points
//
// The status reconciler integrates with:
//   - Kubernetes API server for Deployment monitoring
//   - Control plane via gRPC for status reporting
//   - Circuit breaker pattern for resilience
//   - controller-runtime for event handling and lifecycle
//
// # Error Handling
//
// The reconciler handles different failure modes:
//   - Deployment not found: Treats as 0 available replicas
//   - Control plane errors: Handled through circuit breaker with fallback
//   - Network failures: Automatic retry with exponential backoff
package status

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// StatusReconciler monitors sentinel Deployment status and reports to control plane.
//
// This struct implements controller-runtime.Reconciler to watch Deployment
// resources associated with sentinel components. When deployments change,
// the reconciler extracts operational metrics and reports them to the
// control plane for centralized monitoring and observability.
type StatusReconciler struct {
	// client provides access to Kubernetes API for resource operations.
	client client.Client
	// logger provides structured logging for operations and debugging.
	logger logging.Logger

	// cluster provides access to control plane gRPC APIs for status reporting.
	cluster ctrlv1connect.ClusterServiceClient
	// cb provides circuit breaker protection for control plane communication.
	cb circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.UpdateSentinelStateResponse]]
}

// Config holds the configuration required to create a new StatusReconciler.
//
// All fields are required for the reconciler to function properly.
// The reconciler needs access to the Kubernetes API, control plane,
// and controller runtime manager to monitor and report status.
type Config struct {
	// Client provides access to the Kubernetes API for resource monitoring.
	Client client.Client
	// Logger provides structured logging for operations and debugging.
	Logger logging.Logger
	// Cluster provides access to control plane APIs for status reporting.
	Cluster ctrlv1connect.ClusterServiceClient
	// Manager is the controller-runtime manager that will host this controller.
	Manager manager.Manager
}

// New creates a new StatusReconciler with the provided configuration.
//
// This function initializes the status reconciler and registers it with the
// controller-runtime manager. The reconciler is configured to watch only
// Deployment resources that are labeled as sentinel components, filtering
// out other deployments to reduce unnecessary processing.
//
// The controller includes a circuit breaker to protect against control
// plane unavailability and ensure the controller remains responsive.
//
// Returns an error if the controller cannot be registered with the manager.
func New(cfg Config) (*StatusReconciler, error) {
	r := &StatusReconciler{
		client:  cfg.Client,
		logger:  cfg.Logger,
		cluster: cfg.Cluster,
		cb:      circuitbreaker.New[*connect.Response[ctrlv1.UpdateSentinelStateResponse]]("sentinel_controller_status"),
	}

	err := ctrl.NewControllerManagedBy(cfg.Manager).
		For(
			&appsv1.Deployment{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					return k8s.IsComponentSentinel(obj.GetLabels())
				}),
			),
		).Complete(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Reconcile handles status updates for sentinel Deployment resources.
//
// This method implements the controller-runtime reconcile loop for Deployment
// resources associated with sentinel components. It extracts the current
// operational status and reports it to the control plane.
//
// The reconciliation process:
//  1. Fetch the Deployment resource to check its current status
//  2. Extract available replica count (0 if deployment not found)
//  3. Report status to control plane via gRPC with circuit breaker protection
//  4. Request periodic requeue to ensure continued monitoring
//
// Parameters:
//   - ctx: Context for the reconciliation operation
//   - req: Reconciliation request with namespace and name
//
// Returns ctrl.Result with 15-minute requeue interval and any error encountered.
func (r *StatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	dpl := &appsv1.Deployment{}
	err := r.client.Get(ctx, req.NamespacedName, dpl)

	if err != nil {
		if apierrors.IsNotFound(err) {
			updateErr := r.updateState(ctx, req.Name, 0)
			if updateErr != nil {
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	updateErr := r.updateState(ctx, req.Name, dpl.Status.AvailableReplicas)
	if updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Minute}, nil
}

func (r *StatusReconciler) updateState(ctx context.Context, name string, availableReplicas int32) error {

	_, err := r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		return r.cluster.UpdateSentinelState(ctx, connect.NewRequest(&ctrlv1.UpdateSentinelStateRequest{
			K8SName:           name,
			AvailableReplicas: availableReplicas,
		}))
	})
	return err
}
