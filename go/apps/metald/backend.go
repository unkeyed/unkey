package metald

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/docker"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/kubernetes"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/config"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// convertToInternalConfig converts the external config to internal config format
func convertToInternalConfig(cfg *Config) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Address: cfg.Server.Address,
			Port:    cfg.Server.Port,
		},
		Backend: config.BackendConfig{
			Type: types.BackendType(cfg.Backend.Type),
			Jailer: config.JailerConfig{
				UID:           cfg.Backend.Jailer.UID,
				GID:           cfg.Backend.Jailer.GID,
				ChrootBaseDir: cfg.Backend.Jailer.ChrootBaseDir,
			},
		},
		Database: config.DatabaseConfig{
			DataDir: cfg.Database.DataDir,
		},
		AssetManager: config.AssetManagerConfig{
			Enabled:  cfg.AssetManager.Enabled,
			Endpoint: cfg.AssetManager.Endpoint,
		},
		Billing: config.BillingConfig{
			Enabled:  cfg.Billing.Enabled,
			Endpoint: cfg.Billing.Endpoint,
			MockMode: cfg.Billing.MockMode,
		},
		TLS: &config.TLSConfig{
			Mode:              cfg.TLS.Mode,
			CertFile:          cfg.TLS.CertFile,
			KeyFile:           cfg.TLS.KeyFile,
			CAFile:            cfg.TLS.CAFile,
			SPIFFESocketPath:  cfg.TLS.SPIFFESocketPath,
			EnableCertCaching: cfg.TLS.EnableCertCaching,
			CertCacheTTL:      cfg.TLS.CertCacheTTL,
		},
		OpenTelemetry: config.OpenTelemetryConfig{
			Enabled:                      cfg.OpenTelemetry.Enabled,
			ServiceName:                  cfg.OpenTelemetry.ServiceName,
			ServiceVersion:               cfg.OpenTelemetry.ServiceVersion,
			TracingSamplingRate:          cfg.OpenTelemetry.TracingSamplingRate,
			OTLPEndpoint:                 cfg.OpenTelemetry.OTLPEndpoint,
			PrometheusEnabled:            cfg.OpenTelemetry.PrometheusEnabled,
			PrometheusPort:               cfg.OpenTelemetry.PrometheusPort,
			PrometheusInterface:          cfg.OpenTelemetry.PrometheusInterface,
			HighCardinalityLabelsEnabled: cfg.OpenTelemetry.HighCardinalityLabelsEnabled,
		},
	}
}

// initializeK8sBackend creates a Kubernetes backend
func initializeK8sBackend(ctx context.Context, cfg *Config, logger *slog.Logger) (types.Backend, error) {
	// Create logging.Logger from slog.Logger
	loggingLogger := logging.New().With("backend", "kubernetes")

	backend, err := kubernetes.New(loggingLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kubernetes backend: %w", err)
	}

	return backend, nil
}

// initializeDockerBackend creates a docker backend
func initializeDockerBackend(ctx context.Context, cfg *Config, logger *slog.Logger) (types.Backend, error) {
	// Create logging.Logger from slog.Logger
	loggingLogger := logging.New().With("backend", "docker")

	backend, err := docker.New(loggingLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Docker backend: %w", err)
	}

	return backend, nil
}
