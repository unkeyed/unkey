package konsume

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	clickhousestorage "github.com/unkeyed/unkey/go/pkg/analytics/storage/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"github.com/unkeyed/unkey/go/pkg/version"
)

func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	shutdowns := shutdown.New()

	// Initialize clock
	clk := clock.New()

	// Initialize OpenTelemetry if enabled
	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "konsume",
			Version:         version.Version,
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

	// Initialize logger with context
	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
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

	logger.Info("starting konsume analytics ingestion pipeline",
		"consumer_group", cfg.ConsumerGroup,
		"brokers", cfg.KafkaBrokers,
	)

	// Catch any panics now after we have a logger but before we start the consumers
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	// Initialize Prometheus metrics if enabled
	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}
		go func() {
			promListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
			if err != nil {
				panic(err)
			}
			promListenErr := prom.Serve(ctx, promListener)
			if promListenErr != nil {
				panic(promListenErr)
			}
		}()
	}

	// Initialize database
	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	shutdowns.Register(database.Close)
	logger.Info("database initialized")

	// Initialize vault
	vaultStorage, err := storage.NewS3(storage.S3Config{
		Logger:            logger,
		S3URL:             cfg.VaultS3.S3URL,
		S3Bucket:          cfg.VaultS3.S3Bucket,
		S3AccessKeyID:     cfg.VaultS3.S3AccessKeyID,
		S3AccessKeySecret: cfg.VaultS3.S3AccessKeySecret,
	})
	if err != nil {
		return fmt.Errorf("failed to create vault storage: %w", err)
	}

	vaultService, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    vaultStorage,
		MasterKeys: cfg.VaultMasterKeys,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vault: %w", err)
	}
	logger.Info("vault initialized")

	// Create the Unkey analytics writer (primary writer for all events)
	unkeyWriter, err := clickhousestorage.New(clickhousestorage.Config{
		URL: cfg.ClickhouseURL,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to create clickhouse analytics writer: %w", err)
	}
	shutdowns.RegisterCtx(unkeyWriter.Close)
	logger.Info("unkey analytics writer initialized")

	// Create workspace writer manager
	writerManager, err := NewWorkspaceWriterManager(WorkspaceWriterManagerConfig{
		UnkeyWriter: unkeyWriter,
		DB:          database,
		Vault:       vaultService,
		Clock:       clk,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create workspace writer manager: %w", err)
	}
	shutdowns.RegisterCtx(writerManager.Close)
	logger.Info("workspace writer manager initialized")

	// Create workspace-aware Kafka consumer
	consumer, err := NewWorkspaceAwareConsumer(WorkspaceAwareConsumerConfig{
		Brokers:       cfg.KafkaBrokers,
		Topics:        cfg.Topics,
		InstanceID:    cfg.InstanceID,
		WriterManager: writerManager,
		Logger:        logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create workspace-aware kafka consumer: %w", err)
	}

	// Start consuming
	err = consumer.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start kafka consumer: %w", err)
	}

	// Wait for shutdown signal and gracefully shutdown
	if err := shutdowns.WaitForSignal(ctx, 30*time.Second); err != nil {
		logger.Error("graceful shutdown failed", "error", err.Error())
		return err
	}

	// Stop the consumer
	if stopErr := consumer.Stop(ctx); stopErr != nil {
		logger.Error("failed to stop consumer", "error", stopErr.Error())
		return stopErr
	}

	logger.Info("konsume shut down successfully")
	return nil
}
