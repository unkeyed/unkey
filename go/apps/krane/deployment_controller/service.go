package deploymentcontroller

import (
	"context"
	"fmt"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// DeploymentController manages Deployment resources in Kubernetes.
type DeploymentController struct {
	logger logging.Logger
	client client.Client
	scheme *runtime.Scheme

	events  *buffer.Buffer[*ctrlv1.DeploymentEvent]
	changes *buffer.Buffer[*ctrlv1.UpdateInstanceRequest]
}

var _ k8s.Reconciler = (*DeploymentController)(nil)

// Config holds configuration for creating a DeploymentController.
type Config struct {
	// Logger for Kubernetes operations and debugging.
	Logger logging.Logger
	// Scheme is the Kubernetes runtime scheme for type registration.
	Scheme *runtime.Scheme
	// Client provides access to Kubernetes API.
	Client client.Client

	Manager manager.Manager
}

// New creates a new DeploymentController with the provided configuration.
//
// This function initializes the controller and starts the routing goroutine
// for processing deployment events. The controller will begin handling
// deployment events immediately.
//
// Returns an error if the controller cannot be created.
func New(cfg Config) (*DeploymentController, error) {

	if err := apiv1.AddToScheme(cfg.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add api v1 to scheme: %w", err)
	}

	ctrlruntime.SetLogger(k8s.CompatibleLogger(cfg.Logger))

	c := &DeploymentController{
		logger: cfg.Logger,
		client: cfg.Client,
		scheme: cfg.Scheme,
		events: buffer.New[*ctrlv1.DeploymentEvent](buffer.Config{
			Capacity: 1000,
			Drop:     true,
			Name:     "krane_deployment_events",
		}),
		changes: buffer.New[*ctrlv1.UpdateInstanceRequest](buffer.Config{
			Capacity: 1000,
			Drop:     true,
			Name:     "krane_deployment_changes",
		}),
	}

	if err := ctrlruntime.NewControllerManagedBy(cfg.Manager).
		For(&apiv1.Deployment{}).
		Owns(&appsv1.Deployment{}).
		Named("deployment").
		Complete(c); err != nil {
		return nil, err
	}

	if err := ctrl.NewControllerManagedBy(cfg.Manager).
		For(
			&corev1.Pod{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					return k8s.IsComponentDeployment(obj.GetLabels())
				}),
			),
		).
		Named("deployment_pods2").
		Complete(&podWatcher{
			client:  cfg.Client,
			changes: c.changes,
		}); err != nil {
		return nil, err
	}

	go func() {
		for event := range c.events.Consume() {
			c.logger.Info("Received deployment event", "event", event)
			switch e := event.Event.(type) {
			case *ctrlv1.DeploymentEvent_Apply:
				if err := c.ApplyDeployment(context.Background(), e.Apply); err != nil {
					c.logger.Error("unable to apply deployment", "error", err.Error(), "event", e)
				}
			case *ctrlv1.DeploymentEvent_Delete:
				if err := c.DeleteDeployment(context.Background(), e.Delete); err != nil {
					c.logger.Error("unable to delete deployment", "error", err.Error(), "event", e)
				}
			default:
				c.logger.Error("Unknown deployment event", "event", e)
			}
		}
	}()

	return c, nil
}
