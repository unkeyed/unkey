package sentinelcontroller

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/controlplane"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/reconciler"
	"github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/reflector"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// SentinelController orchestrates sentinel resource management in Kubernetes.
//
// This controller coordinates the complete lifecycle of Sentinel resources
// which provide monitoring, routing, and security functions for Unkey
// deployments. Unlike traditional Kubernetes controllers that watch API events,
// this controller uses a hybrid approach combining control plane streaming
// with standard Kubernetes reconciliation.
//
// The SentinelController serves as the main entry point that coordinates
// three key components:
//   - Reflector: Bridges control plane events to Kubernetes resources
//   - Reconciler: Manages Kubernetes resource lifecycle
//   - Status reporting: Provides operational visibility back to control plane
type SentinelController struct {
	logger logging.Logger
}

// Config holds the configuration required to create a SentinelController.
//
// All fields are required for the controller to function properly. The controller
// needs access to Kubernetes API, control plane connectivity, and identification
// information to properly coordinate sentinel management.
type Config struct {
	// Logger provides structured logging for Kubernetes operations and debugging.
	Logger logging.Logger
	// Scheme is the Kubernetes runtime scheme for type registration.
	Scheme *runtime.Scheme
	// Client provides access to the Kubernetes API for resource management.
	Client client.Client
	// Manager is the controller-runtime manager that hosts the sub-controllers.
	Manager manager.Manager
	// Cluster provides access to control plane APIs for state synchronization.
	Cluster ctrlv1connect.ClusterServiceClient
	// InstanceID uniquely identifies this krane agent instance.
	InstanceID string
	// Region specifies the geographical region this controller operates in.
	Region string
	// Shard specifies the shard identifier for distributed operation.
	Shard string
}

// New creates a new SentinelController with the provided configuration.
//
// This function initializes the complete sentinel management system by setting up
// the required Kubernetes types, configuring logging, and launching the sub-controllers
// that handle different aspects of sentinel lifecycle management.
//
// The initialization process:
//  1. Registers Sentinel API types with the Kubernetes scheme
//  2. Configures controller-runtime logging with krane-compatible logger
//  3. Creates and starts the reflector for control plane event processing
//  4. Creates the reconciler for Kubernetes resource management
//  5. Returns the controller instance for lifecycle management
//
// The reflector runs in a background goroutine and processes control plane
// events continuously. The reconciler is registered with the controller-runtime
// manager and handles standard Kubernetes reconciliation patterns.
//
// Parameters:
//   - cfg: Complete configuration for all controller components
//
// Returns an initialized SentinelController and any error encountered during setup.
// If initialization fails, no background goroutines will be started.
func New(cfg Config) (*SentinelController, error) {
	if err := apiv1.AddToScheme(cfg.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add api v1 to scheme: %w", err)
	}

	ctrlruntime.SetLogger(k8s.CompatibleLogger(cfg.Logger))

	c := &SentinelController{
		logger: cfg.Logger,
	}

	ref := reflector.New(reflector.Config{
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
	go ref.Start(context.Background())

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
