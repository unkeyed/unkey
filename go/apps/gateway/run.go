package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"github.com/unkeyed/unkey/go/pkg/version"
)

// Run starts the Gateway server
// Gateway is a per-tenant gateway service that:
// - Receives requests from Ingress (via mTLS/SPIRE)
// - Executes customer workloads/code
// - Handles API key verification and rate limiting
// - Isolated per workspace/deployment
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New()
	if cfg.GatewayID != "" {
		logger = logger.With(slog.String("gatewayID", cfg.GatewayID))
	}

	if cfg.DeploymentID != "" {
		logger = logger.With(slog.String("deploymentID", cfg.DeploymentID))
	}

	if cfg.WorkspaceID != "" {
		logger = logger.With(slog.String("workspaceID", cfg.WorkspaceID))
	}

	if cfg.Platform != "" {
		logger = logger.With(slog.String("platform", cfg.Platform))
	}

	if cfg.Region != "" {
		logger = logger.With(slog.String("region", cfg.Region))
	}

	if version.Version != "" {
		logger = logger.With(slog.String("version", version.Version))
	}

	// Catch any panics now after we have a logger but before we start the server
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	shutdowns := shutdown.New()
	clk := clock.New()

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(
			ctx,
			otel.Config{
				Application:     "gateway",
				Version:         version.Version,
				InstanceID:      cfg.GatewayID,
				CloudRegion:     cfg.Region,
				TraceSampleRate: cfg.OtelTraceSamplingRate,
			},
			shutdowns,
		)

		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		go func() {
			var promListener net.Listener
			promListener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
			if err != nil {
				panic(err)
			}
			promListenErr := prom.Serve(ctx, promListener)
			if promListenErr != nil {
				panic(promListenErr)
			}
		}()
	}

	var vaultSvc *vault.Service
	if len(cfg.VaultMasterKeys) > 0 && cfg.VaultS3 != nil {
		var vaultStorage storage.Storage
		vaultStorage, err = storage.NewS3(storage.S3Config{
			Logger:            logger,
			S3URL:             cfg.VaultS3.S3URL,
			S3Bucket:          cfg.VaultS3.S3Bucket,
			S3AccessKeyID:     cfg.VaultS3.S3AccessKeyID,
			S3AccessKeySecret: cfg.VaultS3.S3AccessKeySecret,
		})
		if err != nil {
			return fmt.Errorf("unable to create vault storage: %w", err)
		}

		vaultSvc, err = vault.New(vault.Config{
			Logger:     logger,
			Storage:    vaultStorage,
			MasterKeys: cfg.VaultMasterKeys,
		})
		if err != nil {
			return fmt.Errorf("unable to create vault service: %w", err)
		}
	}

	partitionedDB, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create partitioned db: %w", err)
	}
	shutdowns.Register(partitionedDB.Close)

	// Create separate non-partitioned database connection for keys service
	var mainDB db.Database
	mainDB, err = db.New(db.Config{
		PrimaryDSN:  cfg.MainDatabasePrimary,
		ReadOnlyDSN: cfg.MainDatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create main db: %w", err)
	}
	shutdowns.Register(mainDB.Close)

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickhouseURL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL:    cfg.ClickhouseURL,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
	}

	// Use in-memory counter since Redis is nil
	ctr, err := counter.NewRedis(counter.RedisConfig{
		Logger:   logger,
		RedisURL: cfg.RedisURL,
	})
	if err != nil {
		return fmt.Errorf("unable to create counter: %w", err)
	}
	shutdowns.Register(ctr.Close)

	// Create rate limiting service
	rlSvc, err := ratelimit.New(ratelimit.Config{
		Logger:  logger,
		Clock:   clk,
		Counter: ctr,
	})
	if err != nil {
		return fmt.Errorf("unable to create ratelimit service: %w", err)
	}
	shutdowns.Register(rlSvc.Close)

	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		Logger:  logger,
		DB:      mainDB,
		Counter: ctr,
		TTL:     30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("unable to create usage limiter service: %w", err)
	}

	// Create key service with non-partitioned database
	keySvc, err := keys.New(keys.Config{
		Logger:       logger,
		DB:           mainDB,
		KeyCache:     nil, // TODO: Initialize caches
		RateLimiter:  rlSvc,
		RBAC:         rbac.New(),
		UsageLimiter: ulSvc,
		Clickhouse:   ch,
		Region:       cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("unable to create key service: %w", err)
	}
	shutdowns.Register(keySvc.Close)

	// TODO: Initialize SPIRE client if enabled
	// TODO: Initialize customer workload execution service
	// TODO: Create HTTP/HTTPS servers

	logger.Info("Gateway server initialized",
		"deploymentID", cfg.DeploymentID,
		"workspaceID", cfg.WorkspaceID,
		"region", cfg.Region,
		"keySvc", keySvc,
		"vault", vaultSvc,
	)

	// Wait for either OS signals or context cancellation, then shutdown
	if err := shutdowns.WaitForSignal(ctx, 0); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Gateway server shut down successfully")
	return nil
}
