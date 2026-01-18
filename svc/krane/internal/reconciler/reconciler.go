package reconciler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"k8s.io/client-go/kubernetes"
)

// Reconciler synchronizes control plane deployment state with Kubernetes resources.
//
// The reconciler maintains bidirectional state synchronization: it receives desired
// state from the control plane via [HandleState] and applies it to Kubernetes, then
// watches Kubernetes for actual state changes and reports them back. This dual-flow
// ensures the control plane always knows what's actually running in the cluster.
//
// A single Reconciler manages all deployments and sentinels in its cluster. It uses
// background goroutines for watching and refreshing, so callers must call [Start]
// before processing state and [Stop] during shutdown.
type Reconciler struct {
	clientSet kubernetes.Interface
	logger    logging.Logger
	cluster   ctrlv1connect.ClusterServiceClient
	cb        circuitbreaker.CircuitBreaker[any]
	done      chan struct{}
	clusterID string
	region    string
	// last seen sequence
	sequence uint64
}

// Config holds the configuration required to create a new [Reconciler].
// All fields are required.
type Config struct {
	ClientSet kubernetes.Interface
	Logger    logging.Logger
	Cluster   ctrlv1connect.ClusterServiceClient
	ClusterID string
	Region    string
}

// New creates a [Reconciler] ready to be started with [Reconciler.Start].
func New(cfg Config) *Reconciler {
	return &Reconciler{
		clientSet: cfg.ClientSet,
		logger:    cfg.Logger,
		cluster:   cfg.Cluster,
		cb:        circuitbreaker.New[any]("reconciler_state_update"),
		done:      make(chan struct{}),
		clusterID: cfg.ClusterID,
		region:    cfg.Region,
		sequence:  0,
	}
}

// Start launches the background watch and refresh loops for Kubernetes resources.
//
// The watch loops provide real-time state updates when resources change. The refresh
// loops run every minute to catch missed events and ensure eventual consistency.
// Both loops continue until the context is cancelled or [Reconciler.Stop] is called.
func (r *Reconciler) Start(ctx context.Context) error {
	go r.refreshCurrentDeployments(ctx)
	go r.refreshCurrentSentinels(ctx)

	if err := r.watchCurrentSentinels(ctx); err != nil {
		return err
	}
	if err := r.watchCurrentDeployments(ctx); err != nil {
		return err
	}

	stream, err := r.cluster.Sync(ctx, connect.NewRequest(&ctrlv1.SyncRequest{
		ClusterId: r.clusterID,
		Region:    r.region,
	}))
	if err != nil {
		return err
	}

	for stream.Receive() {
		if err := r.HandleState(ctx, stream.Msg()); err != nil {
			r.logger.Error("error handling state", "error", err)
		}
	}
	err = stream.Close()
	if err != nil {
		r.logger.Error("unable to close stream", "error", err)
	}

	go r.Watch(ctx)

	return nil
}

// HandleState applies a single state update from the control plane to the cluster.
//
// The state contains either a deployment or sentinel operation (apply or delete).
// For apply operations, HandleState creates or updates the Kubernetes resource and
// reports the resulting state back to the control plane. For delete operations, it
// removes the resource and confirms deletion.
//
// HandleState returns immediately after processing the single state update. It does
// not block waiting for additional updates; use it within a loop that reads from
// the control plane's state stream.
func (r *Reconciler) HandleState(ctx context.Context, state *ctrlv1.State) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}
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

	r.sequence = state.GetSequence()
	return nil
}

// Stop signals all background goroutines to terminate. Safe to call multiple
// times, but not concurrently with itself.
func (r *Reconciler) Stop() error {
	close(r.done)
	return nil
}
