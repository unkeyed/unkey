package krane

import (
	"context"
	"fmt"
	"log/slog"

	deploymentcontroller "github.com/unkeyed/unkey/go/apps/krane/deployment_controller"
	gatewaycontroller "github.com/unkeyed/unkey/go/apps/krane/gateway_controller"
	"github.com/unkeyed/unkey/go/apps/krane/sync"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
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

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "krane",
			Version:         pkgversion.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		},
			shutdowns,
		)
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

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

	s, err := sync.New(sync.Config{
		Logger:             logger,
		Region:             cfg.Region,
		InstanceID:         cfg.InstanceID,
		ControlPlaneURL:    cfg.ControlPlaneURL,
		ControlPlaneBearer: cfg.ControlPlaneBearer,
	})
	if err != nil {
		return fmt.Errorf("failed to create sync engine: %w", err)
	}

	dc, err := deploymentcontroller.New(deploymentcontroller.Config{
		Logger: logger,
		Buffer: s.DeploymentUpdateBuffer,
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment controller: %w", err)
	}

	gc, err := gatewaycontroller.New(gatewaycontroller.Config{
		Logger: logger,
		Buffer: s.GatewayUpdateBuffer,
	})
	if err != nil {
		return fmt.Errorf("failed to create gateway controller: %w", err)
	}

	go func() {
		for e := range s.Subscribe() {
			ctx := context.Background()
			logger.Info("Event received", "event", e)

			var backendErr error
			switch x := e.Event.(type) {
			case *ctrlv1.InfraEvent_DeploymentEvent:
				{
					switch y := x.DeploymentEvent.Event.(type) {
					case *ctrlv1.DeploymentEvent_Apply:
						backendErr = dc.ApplyDeployment(ctx, y.Apply)
					case *ctrlv1.DeploymentEvent_Delete:
						backendErr = dc.DeleteDeployment(ctx, y.Delete)
					}

				}
				switch x := e.Event.(type) {
				case *ctrlv1.InfraEvent_GatewayEvent:
					{
						switch y := x.GatewayEvent.Event.(type) {
						case *ctrlv1.GatewayEvent_Apply:
							backendErr = gc.ApplyGateway(ctx, y.Apply)
						case *ctrlv1.GatewayEvent_Delete:
							backendErr = gc.DeleteGateway(ctx, y.Delete)
						}

					}
				}

			default:
				logger.Warn("Unknown event type", "event", x)
			}
			if backendErr != nil {
				logger.Error("unable to enact infra event", "error", backendErr)
			}
		}
	}()

	// Wait for signal and handle shutdown
	logger.Info("Krane server started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("krane server shut down successfully")
	return nil
}
