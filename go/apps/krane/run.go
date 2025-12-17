package krane

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"k8s.io/client-go/kubernetes/scheme"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/controlplane"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	sentinelcontroller "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	pkgversion "github.com/unkeyed/unkey/go/pkg/version"
)

// Run starts the krane agent with the provided configuration.
//
// This function initializes all necessary components for the krane agent including
// Kubernetes controllers, synchronization engine, and observability infrastructure.
// It establishes connections to the control plane and begins the reconciliation
// loop to maintain desired state.
//
// The function performs the following initialization steps:
//  1. Validates the configuration using [Config.Validate]
//  2. Sets up structured logging with instance metadata
//  3. Creates buffered channels for event streaming
//  4. Initializes Kubernetes client and controller manager
//  5. Creates deployment and sentinel controllers
//  6. Sets up the synchronization engine with control plane connection
//  7. Starts Prometheus metrics server if configured
//  8. Blocks until context cancellation for graceful shutdown
//
// The function handles graceful shutdown by registering all components with the
// shutdown manager and waiting for signal termination. All resources are properly
// cleaned up before the function returns.
//
// Parameters:
//   - ctx: Context for controlling the agent lifecycle and cancellation
//   - cfg: Configuration containing connection details, credentials, and settings
//
// Returns an error if initialization fails or if shutdown encounters problems.
// Returns nil on successful graceful shutdown.
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	shutdowns := shutdown.New()

	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}
	if cfg.Region != "" {
		logger = logger.With(slog.String("region", cfg.Region))
	}
	if pkgversion.Version != "" {
		logger = logger.With(slog.String("version", pkgversion.Version))
	}

	cluster := controlplane.NewClient(controlplane.ClientConfig{
		URL:         cfg.ControlPlaneURL,
		BearerToken: cfg.ControlPlaneBearer,
		Region:      cfg.Region,
		Shard:       cfg.Shard,
	})

	client, err := k8s.NewClient(scheme.Scheme)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	manager, err := k8s.NewManager(scheme.Scheme)
	if err != nil {
		return fmt.Errorf("failed to create k8s manager: %w", err)
	}

	sc, err := sentinelcontroller.New(sentinelcontroller.Config{
		Logger:     logger,
		Scheme:     scheme.Scheme,
		Client:     client,
		Manager:    manager,
		Cluster:    cluster,
		InstanceID: cfg.InstanceID,
		Region:     cfg.Region,
		Shard:      cfg.Shard,
	})
	if err != nil {
		return fmt.Errorf("failed to create sentinel controller: %w", err)
	}
	var _ = sc

	go func() {

		err = manager.Start(ctx)
		if err != nil {
			logger.Error("failed to start k8s manager", "error", err)
		}
	}()

	if cfg.PrometheusPort > 0 {

		prom, err := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create prometheus server: %w", err)
		}

		shutdowns.RegisterCtx(prom.Shutdown)
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if err != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, err)
		}
		go func() {
			logger.Info("prometheus started", "port", cfg.PrometheusPort)
			if err := prom.Serve(ctx, ln); err != nil {
				logger.Error("failed to start prometheus server", "error", err)
			}
		}()

	}
	// Wait for signal and handle shutdown
	logger.Info("Krane server started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("krane server shut down successfully")
	return nil
}
