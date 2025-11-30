package krane

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
	"github.com/unkeyed/unkey/go/apps/krane/backend/docker"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes"
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

	var b backend.Backend
	switch cfg.Backend {
	case Docker:
		b, err = docker.New(docker.Config{
			Logger:           logger,
			SocketPath:       cfg.DockerSocketPath,
			RegistryURL:      cfg.RegistryURL,
			RegistryUsername: cfg.RegistryUsername,
			RegistryPassword: cfg.RegistryPassword,
		})
		if err != nil {
			return fmt.Errorf("failed to create docker backend: %w", err)
		}
	case Kubernetes:
		b, err = kubernetes.New(kubernetes.Config{
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create kubernetes backend: %w", err)
		}
	default:
		return fmt.Errorf("unsupported backend: %s", cfg.Backend)
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

	dm := deploymentManager{
		logger:  logger,
		backend: b,
	}

	gm := gatewayManager{
		logger:  logger,
		backend: b,
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
						backendErr = dm.HandleApply(ctx, y.Apply)
					case *ctrlv1.DeploymentEvent_Delete:
						backendErr = dm.HandleDelete(ctx, y.Delete)
					}

				}
				switch x := e.Event.(type) {
				case *ctrlv1.InfraEvent_GatewayEvent:
					{
						switch y := x.GatewayEvent.Event.(type) {
						case *ctrlv1.GatewayEvent_Apply:
							backendErr = gm.HandleApply(ctx, y.Apply)
						case *ctrlv1.GatewayEvent_Delete:
							backendErr = gm.HandleDelete(ctx, y.Delete)
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
