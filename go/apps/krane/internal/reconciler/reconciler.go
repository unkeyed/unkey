package reconciler

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"k8s.io/client-go/kubernetes"
)

// Reconciler watches control plane changes and reflects them in the Kubernetes cluster.
//
// This struct bridges the gap between the control plane's deployment state
// and Kubernetes resources by processing streaming events and applying
// them as custom resources. It maintains consistency between the database
// and cluster state through event processing and periodic refresh.
type Reconciler struct {
	clientSet *kubernetes.Clientset
	logger    logging.Logger

	// cluster provides access to control plane APIs for state queries
	cluster ctrlv1connect.ClusterServiceClient
	cb      circuitbreaker.CircuitBreaker[any]
	done    chan struct{}
	region  string
}

// Config holds the configuration required to create a new Reconciler.
//
// All fields are required for the Reconciler to function properly.
// The Reconciler needs access to the Kubernetes API, logging, and the
// control plane streaming interface to maintain synchronization.
type Config struct {
	// Client provides access to the Kubernetes API for resource management.
	ClientSet *kubernetes.Clientset
	// Logger provides structured logging for operations and debugging.
	Logger logging.Logger
	// Cluster provides access to control plane APIs for state queries.
	Cluster ctrlv1connect.ClusterServiceClient

	ClusterID string
	Region    string
}

// New creates a new Reconciler with the provided configuration.
//
// This function initializes the Reconciler with all required dependencies.
// The returned Reconciler is ready to start processing events after
// calling the Start method.
func New(cfg Config) (*Reconciler, error) {
	r := &Reconciler{
		clientSet: cfg.ClientSet,
		logger:    cfg.Logger,
		cluster:   cfg.Cluster,
		cb:        circuitbreaker.New[any]("reconciler_state_update"),
		done:      make(chan struct{}),
		region:    cfg.Region,
	}

	go r.refreshCurrentDeployments(context.Background())
	go r.refreshCurrentSentinels(context.Background())

	if err := r.watchCurrentSentinels(context.Background()); err != nil {
		return nil, err
	}
	if err := r.watchCurrentDeployments(context.Background()); err != nil {
		return nil, err
	}

	return r, nil
}

// Start begins the Reconciler's event processing loop.
//
// This method starts the event processing pipeline:
//  1. Creates a buffer for handling high-throughput events
//  2. Initiates control plane streaming for real-time updates
//  3. Starts periodic refresh for consistency assurance
//  4. Processes events from the buffer until context is cancelled
//
// The event processing distinguishes between Apply and Delete operations
// and routes them to appropriate handlers. Unknown event types are logged
// as errors for debugging.
//
// This method blocks until the provided context is cancelled.
func (r *Reconciler) HandleState(ctx context.Context, state *ctrlv1.State) error {
	switch kind := state.GetKind().(type) {
	case *ctrlv1.State_Deployment:
		{
			switch op := kind.Deployment.GetState().(type) {
			case *ctrlv1.DeploymentState_Apply:
				if err := r.ApplyDeployment(ctx, op.Apply); err != nil {
					return err
				}
			case *ctrlv1.DeploymentState_Delete:
				if err := r.DeleteDeployment(ctx, op.Delete); err != nil {
					return err
				}
			}
		}
	case *ctrlv1.State_Sentinel:
		{
			switch op := kind.Sentinel.GetState().(type) {
			case *ctrlv1.SentinelState_Apply:
				if err := r.ApplySentinel(ctx, op.Apply); err != nil {
					return err
				}
			case *ctrlv1.SentinelState_Delete:
				if err := r.DeleteSentinel(ctx, op.Delete); err != nil {
					return err
				}
			}
		}

	default:
		return fmt.Errorf("unknown state type: %T", kind)
	}

	return nil
}

func (r *Reconciler) Stop() error {
	close(r.done)
	return nil
}
