//go:build linux
// +build linux

package metald

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/go/apps/metald/internal/assetmanager"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/firecracker"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/config"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

// initializeFirecrackerBackend creates a firecracker backend (Linux only)
func initializeFirecrackerBackend(ctx context.Context, cfg *Config, logger *slog.Logger, tlsProvider tlspkg.Provider) (types.Backend, error) {
	// Base directory for VM data
	baseDir := "/opt/metald/vms"

	// Create AssetManager client for asset preparation
	var assetClient assetmanager.Client
	var err error

	if cfg.AssetManager.Enabled {
		// Use TLS-enabled HTTP client
		httpClient := tlsProvider.HTTPClient()

		// Convert to internal AssetManager config
		assetCfg := &config.AssetManagerConfig{
			Enabled:  cfg.AssetManager.Enabled,
			Endpoint: cfg.AssetManager.Endpoint,
		}

		assetClient, err = assetmanager.NewClientWithHTTP(assetCfg, logger, httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create assetmanager client: %w", err)
		}
		logger.Info("initialized assetmanager client",
			slog.String("endpoint", cfg.AssetManager.Endpoint),
		)
	} else {
		// Use noop client if assetmanager is disabled
		assetCfg := &config.AssetManagerConfig{
			Enabled: false,
		}
		assetClient, _ = assetmanager.NewClient(assetCfg, logger)
		logger.Info("assetmanager disabled, using noop client")
	}

	// Convert to internal Jailer config
	jailerCfg := &config.JailerConfig{
		UID:           cfg.Backend.Jailer.UID,
		GID:           cfg.Backend.Jailer.GID,
		ChrootBaseDir: cfg.Backend.Jailer.ChrootBaseDir,
	}

	sdkClient, err := firecracker.NewClient(logger, assetClient, jailerCfg, baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create firecracker client: %w", err)
	}

	logger.Info("initialized firecracker backend",
		slog.String("firecracker_binary", "/usr/local/bin/firecracker"),
		slog.Uint64("uid", uint64(cfg.Backend.Jailer.UID)),
		slog.Uint64("gid", uint64(cfg.Backend.Jailer.GID)),
		slog.String("chroot_base", cfg.Backend.Jailer.ChrootBaseDir),
	)

	return sdkClient, nil
}
