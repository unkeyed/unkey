package deploymentreflector

import (
	"context"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/controlplane"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"k8s.io/client-go/kubernetes"
)

// Reflector watches control plane changes and reflects them in the Kubernetes cluster.
//
// This struct bridges the gap between the control plane's deployment state
// and Kubernetes resources by processing streaming events and applying
// them as custom resources. It maintains consistency between the database
// and cluster state through event processing and periodic refresh.
type Reflector struct {
	clientSet *kubernetes.Clientset
	logger    logging.Logger

	// watcher streams deployment state changes from the control plane
	watcher *controlplane.Watcher[ctrlv1.DeploymentState]
	// cluster provides access to control plane APIs for state queries
	cluster ctrlv1connect.ClusterServiceClient
	inbound *buffer.Buffer[*ctrlv1.DeploymentState]
	cb      circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.UpdateDeploymentStateResponse]]
	done    chan struct{}
}

// Config holds the configuration required to create a new Reflector.
//
// All fields are required for the reflector to function properly.
// The reflector needs access to the Kubernetes API, logging, and the
// control plane streaming interface to maintain synchronization.
type Config struct {
	// Client provides access to the Kubernetes API for resource management.
	ClientSet *kubernetes.Clientset
	// Logger provides structured logging for operations and debugging.
	Logger logging.Logger
	// Cluster provides access to control plane APIs for state queries.
	Cluster ctrlv1connect.ClusterServiceClient

	InstanceID string
	Region     string
	Shard      string
}

// New creates a new Reflector with the provided configuration.
//
// This function initializes the reflector with all required dependencies.
// The returned Reflector is ready to start processing events after
// calling the Start method.
func New(cfg Config) (*Reflector, error) {
	r := &Reflector{
		clientSet: cfg.ClientSet,
		logger:    cfg.Logger,
		cluster:   cfg.Cluster,
		watcher: controlplane.NewWatcher(controlplane.WatcherConfig[ctrlv1.DeploymentState]{
			Logger:       cfg.Logger,
			InstanceID:   cfg.InstanceID,
			Region:       cfg.Region,
			Shard:        cfg.Shard,
			CreateStream: cfg.Cluster.WatchDeployments,
		}),

		inbound: buffer.New[*ctrlv1.DeploymentState](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "deployment_controller_inbound",
		}),
		cb:   circuitbreaker.New[*connect.Response[ctrlv1.UpdateDeploymentStateResponse]]("deployment_controller_status"),
		done: make(chan struct{}),
	}

	return r, nil
}

// Start begins the reflector's event processing loop.
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
func (r *Reflector) Start() {

	ctx := context.Background()
	r.watcher.Sync(ctx, r.inbound)
	r.watcher.Watch(ctx, r.inbound)
	r.refreshCurrentReplicaSets(ctx)
	go r.watchCurrentDeployments(ctx)

	for {
		select {
		case <-r.done:
			return
		case e := <-r.inbound.Consume():
			switch state := e.GetState().(type) {
			case *ctrlv1.DeploymentState_Apply:
				if err := r.applyDeployment(ctx, state.Apply); err != nil {
					r.logger.Error("unable to apply deployment", "error", err.Error(), "deployment_id", state.Apply.GetDeploymentId())
				}
			case *ctrlv1.DeploymentState_Delete:
				if err := r.deleteDeployment(ctx, state.Delete); err != nil {
					r.logger.Error("unable to delete deployment", "error", err.Error(), "deployment_k8s_name", state.Delete.GetK8SName())
				}
			default:
				r.logger.Error("Unknown deployment event", "event", state)
			}
		}
	}
}

func (r *Reflector) Stop() error {
	close(r.done)
	return nil
}
