package sentinelcontroller

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/controlplane"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/inbound"
	"github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/reconciler"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

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
}

// Config holds configuration for creating a SentinelController.
type Config struct {
	// Logger for Kubernetes operations and debugging.
	Logger logging.Logger
	// Scheme is the Kubernetes runtime scheme for type registration.
	Scheme *runtime.Scheme
	// Client provides access to Kubernetes API.
	Client client.Client

	Manager manager.Manager

	Cluster ctrlv1connect.ClusterServiceClient

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

	c := &SentinelController{
		logger: cfg.Logger,
	}

	ir := inbound.New(inbound.Config{
		Client:  cfg.Client,
		Logger:  cfg.Logger,
		Cluster: cfg.Cluster,
		Watcher: controlplane.NewWatcher(controlplane.WatcherConfig[ctrlv1.SentinelState]{
			Logger:       cfg.Logger,
			InstanceID:   cfg.InstanceID,
			Region:       cfg.Region,
			Shard:        cfg.Shard,
			CreateStream: cfg.Cluster.WatchSentinels,
		}),
	})
	go ir.Start(context.Background())

	_, err := reconciler.New(reconciler.Config{
		Logger:  cfg.Logger,
		Scheme:  cfg.Scheme,
		Client:  cfg.Client,
		Manager: cfg.Manager,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reconciler: %w", err)
	}

	return c, nil
}
