package reconciler

import (
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

type Reconciler struct {
	logger logging.Logger
	client client.Client
	scheme *runtime.Scheme
}

type Config struct {
	// Logger for Kubernetes operations and debugging.
	Logger logging.Logger
	// Scheme is the Kubernetes runtime scheme for type registration.
	Scheme *runtime.Scheme
	// Client provides access to Kubernetes API.
	Client client.Client

	Manager manager.Manager
}

var _ k8s.Reconciler = (*Reconciler)(nil)

// New creates a new SentinelController with the provided configuration.
//
// This function initializes the controller and starts the routing goroutine
// for processing sentinel events. The controller will begin handling
// sentinel events immediately.
//
// Returns an error if the controller cannot be created.
func New(cfg Config) (*Reconciler, error) {

	r := &Reconciler{
		logger: cfg.Logger,
		client: cfg.Client,
		scheme: cfg.Scheme,
	}

	err := ctrlruntime.NewControllerManagedBy(cfg.Manager).
		For(&apiv1.Sentinel{}).     // nolint:exhaustruct
		Owns(&appsv1.Deployment{}). // nolint:exhaustruct
		Named("sentinel").
		Complete(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
