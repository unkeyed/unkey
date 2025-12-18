package sentinelreflector

import (
	"context"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/controlplane"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Reflector watches control plane changes and reflects them in the Kubernetes cluster.
//
// This struct bridges the gap between the control plane's deployment state
// and Kubernetes resources by processing streaming events and applying
// them as custom resources. It maintains consistency between the database
// and cluster state through event processing and periodic refresh.
type Reflector struct {
	//client    client.Client
	clientSet *kubernetes.Clientset
	logger    logging.Logger

	// watcher streams Sentinel state changes from the control plane
	watcher *controlplane.Watcher[ctrlv1.SentinelState]
	// cluster provides access to control plane APIs for state queries
	cluster ctrlv1connect.ClusterServiceClient
	// inbound is a buffer for incoming Sentinel state updates
	inbound *buffer.Buffer[*ctrlv1.SentinelState]
	cb      circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.UpdateSentinelStateResponse]]

	done chan struct{}
}

// Config holds the configuration required to create a new Reflector.
//
// All fields are required for the reflector to function properly.
// The reflector needs access to the Kubernetes API, logging, and the
// control plane streaming interface to maintain synchronization.
type Config struct {
	// ClientSet provides access to the Kubernetes API for resource management.
	ClientSet *kubernetes.Clientset
	// Logger provides structured logging for operations and debugging.
	Logger logging.Logger
	// Cluster provides access to control plane APIs for state queries.
	Cluster ctrlv1connect.ClusterServiceClient

	Manager manager.Manager

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
		watcher: controlplane.NewWatcher(controlplane.WatcherConfig[ctrlv1.SentinelState]{
			Logger:       cfg.Logger,
			InstanceID:   cfg.InstanceID,
			Region:       cfg.Region,
			Shard:        cfg.Shard,
			CreateStream: cfg.Cluster.WatchSentinels,
		}),
		inbound: buffer.New[*ctrlv1.SentinelState](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "deployment_controller_inbound",
		}),
		cb:   circuitbreaker.New[*connect.Response[ctrlv1.UpdateSentinelStateResponse]]("sentinel_controller_status"),
		done: make(chan struct{}),
	}

	err := controllerruntime.NewControllerManagedBy(cfg.Manager).
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
	r.refreshCurrentDeployments(ctx)

	for {
		select {
		case <-r.done:
			return
		case e := <-r.inbound.Consume():
			switch state := e.GetState().(type) {
			case *ctrlv1.SentinelState_Apply:
				if err := r.applySentinel(ctx, state.Apply); err != nil {
					r.logger.Error("unable to apply sentinel", "error", err.Error(), "sentinel_id", state.Apply.GetSentinelId())
				}
			case *ctrlv1.SentinelState_Delete:
				if err := r.deleteSentinel(ctx, state.Delete); err != nil {
					r.logger.Error("unable to delete Sentinel", "error", err.Error(), "Sentinel_k8s_name", state.Delete.GetK8SName())
				}
			default:
				r.logger.Error("Unknown Sentinel event", "event", state)
			}
		}
	}
}

func (r *Reflector) Stop() error {
	close(r.done)
	return nil
}
