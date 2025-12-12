package krane

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	deploymentcontroller "github.com/unkeyed/unkey/go/apps/krane/deployment_controller"
	gatewaycontroller "github.com/unkeyed/unkey/go/apps/krane/gateway_controller"
	"github.com/unkeyed/unkey/go/apps/krane/sync"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	pkgversion "github.com/unkeyed/unkey/go/pkg/version"
)

// Run starts the krane server with the provided configuration.
//
// Initializes the selected backend (Docker or Kubernetes), sets up HTTP/2
// server with Connect protocol, and handles graceful shutdown on context
// cancellation.
//
// When cfg.OtelEnabled is true, initializes OpenTelemetry tracing, metrics,
// and logging integration.
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

	gatewayUpdates := buffer.New[*ctrlv1.UpdateGatewayRequest](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "krane_gateway_updates",
	})
	gatewayEvents := buffer.New[*ctrlv1.GatewayEvent](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "krane_gateway_events",
	})

	deploymentEvents := buffer.New[*ctrlv1.DeploymentEvent](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "krane_deployment_events",
	})

	instanceUpdates := buffer.New[*ctrlv1.UpdateInstanceRequest](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "krane_instance_updates",
	})

	dc, err := deploymentcontroller.New(deploymentcontroller.Config{
		Logger:  logger,
		Events:  deploymentEvents,
		Updates: instanceUpdates,
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment controller: %w", err)
	}

	gc, err := gatewaycontroller.New(gatewaycontroller.Config{
		Logger: logger,
		Events: gatewayEvents,
	})
	if err != nil {
		return fmt.Errorf("failed to create gateway controller: %w", err)
	}
	s, err := sync.New(sync.Config{
		Logger:               logger,
		Region:               cfg.Region,
		Shard:                cfg.Shard,
		InstanceID:           cfg.InstanceID,
		ControlPlaneURL:      cfg.ControlPlaneURL,
		ControlPlaneBearer:   cfg.ControlPlaneBearer,
		InstanceUpdates:      instanceUpdates,
		GatewayUpdates:       gatewayUpdates,
		DeploymentController: dc,
		GatewayController:    gc,
	})
	if err != nil {
		return fmt.Errorf("failed to create sync engine: %w", err)
	}

	var _ = s

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
