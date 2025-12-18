package reconciler

import (
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// Reconciler implements the controller-runtime.Reconciler interface for Sentinel resources.
//
// This struct contains the dependencies needed to reconcile Sentinel custom resources
// with their corresponding Kubernetes Deployments and Services. It follows the
// standard controller-runtime patterns and integrates with the krane ecosystem.
type Reconciler struct {
	logger logging.Logger
	client client.Client
	scheme *runtime.Scheme
}

// Config holds the configuration required to create a new Reconciler.
//
// All fields are required for the reconciler to function properly.
// The reconciler needs access to the Kubernetes API, a scheme for type
// registration, and a logger for observability and debugging.
type Config struct {
	// Logger provides structured logging for Kubernetes operations and debugging.
	Logger logging.Logger
	// Scheme is the Kubernetes runtime scheme for type registration.
	Scheme *runtime.Scheme
	// Client provides access to the Kubernetes API for resource management.
	Client client.Client
	// Manager is the controller-runtime manager that will host this controller.
	Manager manager.Manager
}

var _ k8s.Reconciler = (*Reconciler)(nil)

// New creates a new Sentinel reconciler with the provided configuration.
//
// This function initializes the controller and registers it with the
// controller-runtime manager. The reconciler will watch Sentinel resources
// for changes and automatically manage the lifecycle of owned resources
// (Deployments and Services).
//
// The controller is configured to:
//   - Watch Sentinel custom resources for changes
//   - Own and manage Deployment resources created for sentinels
//   - Own and manage Service resources for network exposure
//   - Use the "sentinel" name for controller identification
//
// Returns an error if the controller cannot be registered with the manager.
func New(cfg Config) (*Reconciler, error) {
	r := &Reconciler{
		logger: cfg.Logger,
		client: cfg.Client,
		scheme: cfg.Scheme,
	}

	err := ctrlruntime.NewControllerManagedBy(cfg.Manager).
		For(&apiv1.Sentinel{}).     // nolint:exhaustruct
		Owns(&appsv1.Deployment{}). // nolint:exhaustruct
		Owns(&corev1.Service{}).    // nolint:exhaustruct
		Named("sentinel").
		Complete(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
