package sentinelcontroller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/watcher"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// SentinelController manages Sentinel resources in Kubernetes.
//
// This controller handles the lifecycle of Sentinel resources which provide
// monitoring, routing, and security functions for Unkey deployments. Unlike
// the deployment controller, this controller uses a custom routing mechanism
// instead of the standard controller-runtime pattern.
type SentinelController struct {
	logger logging.Logger
	client client.Client
	scheme *runtime.Scheme
}

var _ k8s.Reconciler = (*SentinelController)(nil)

// Config holds configuration for creating a SentinelController.
type Config struct {
	// Logger for Kubernetes operations and debugging.
	Logger logging.Logger
	// Scheme is the Kubernetes runtime scheme for type registration.
	Scheme *runtime.Scheme
	// Client provides access to Kubernetes API.
	Client client.Client

	Manager manager.Manager

	Cluster    ctrlv1connect.ClusterServiceClient
	InstanceID string
	Region     string
	Shard      string
}

// New creates a new SentinelController with the provided configuration.
//
// This function initializes the controller and starts the routing goroutine
// for processing sentinel events. The controller will begin handling
// sentinel events immediately.
//
// Returns an error if the controller cannot be created.
func New(cfg Config) (*SentinelController, error) {

	if err := apiv1.AddToScheme(cfg.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add api v1 to scheme: %w", err)
	}

	ctrlruntime.SetLogger(k8s.CompatibleLogger(cfg.Logger))

	r := &reconciler{
		logger: cfg.Logger,
		client: cfg.Client,
		scheme: cfg.Scheme,
	}

	push := &pusher{
		client:  cfg.Client,
		cb:      circuitbreaker.New[*connect.Response[ctrlv1.PushSentinelStateResponse]]("krane_sentinel_pusher"),
		cluster: cfg.Cluster,
	}

	if err := ctrlruntime.NewControllerManagedBy(cfg.Manager).
		For(
			&appsv1.Deployment{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					return k8s.IsComponentSentinel(obj.GetLabels())
				}),
			),
		).
		Named("sentinel_deployments").
		Complete(push); err != nil {
		return nil, err
	}

	c := &SentinelController{
		logger: cfg.Logger,
		client: cfg.Client,
		scheme: cfg.Scheme,
	}

	if err := ctrlruntime.NewControllerManagedBy(cfg.Manager).
		For(&apiv1.Sentinel{}).
		Owns(&appsv1.Deployment{}).
		Named("sentinel").
		Complete(r); err != nil {
		return nil, err
	}

	w := watcher.New[ctrlv1.SentinelEvent](watcher.Config{
		Logger:     cfg.Logger,
		Cluster:    &cfg.Cluster,
		InstanceID: cfg.InstanceID,
		Region:     cfg.Region,
		Shard:      cfg.Shard,
	})

	w.Sync(context.Background(), cfg.Cluster.PullSentinelState)

	go func() {
		for event := range w.Watch(context.Background(), cfg.Cluster.WatchSentinels) {
			c.logger.Info("Received sentinel event", "event", event)
			switch e := event.Event.(type) {
			case *ctrlv1.SentinelEvent_Apply:
				if err := c.ApplySentinel(context.Background(), e.Apply); err != nil {
					c.logger.Error("unable to apply sentinel", "error", err.Error(), "event", e)
				}
			case *ctrlv1.SentinelEvent_Delete:
				if err := c.DeleteSentinel(context.Background(), e.Delete); err != nil {
					c.logger.Error("unable to delete sentinel", "error", err.Error(), "event", e)
				}
			default:
				c.logger.Error("Unknown sentinel event", "event", e)
			}
		}
	}()

	return c, nil
}
