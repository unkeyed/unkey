package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/apps/controlplane/workflows"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	hydraStore "github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/version"
)

func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}

	shutdowns := shutdown.New()

	// Initialize OpenTelemetry if enabled
	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "controlplane",
			Version:         version.Version,
			InstanceID:      cfg.InstanceID,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
			CloudRegion:     "",
		}, shutdowns)
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}

	}
	// Initialize logger with service context
	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}
	if version.Version != "" {
		logger = logger.With(slog.String("version", version.Version))
	}

	logger.Info("starting controlplane service")

	// Catch any panics after we have a logger
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	store, err := hydraStore.NewMySQLStore(cfg.HydraDatabaseDSN, cfg.Clock)
	if err != nil {
		return err
	}

	// Initialize hydra engine
	engine := hydra.New(hydra.Config{
		Store:      store,
		Clock:      clk,
		Logger:     logger,
		Namespace:  "controlplane",
		Marshaller: nil,
	})

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.UnkeyDatabaseDSN,
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to connect to business database: %w", err)
	}

	ch, err := clickhouse.New(clickhouse.Config{
		URL:    cfg.ClickHouseURL,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to connect to ClickHouse: %w", err)
	}

	// Create hydra worker
	worker, err := hydra.NewWorker(engine, hydra.WorkerConfig{
		WorkerID:          "",
		Concurrency:       10,
		PollInterval:      time.Second,
		HeartbeatInterval: time.Second,
		ClaimTimeout:      5 * time.Second,
		CronInterval:      time.Minute,
	})
	if err != nil {
		return fmt.Errorf("unable to create hydra worker: %w", err)
	}

	quotaChecks := &workflows.QuotaCheckWorkflow{

		Database:     database,
		Clickhouse:   ch,
		Logger:       logger,
		SlackWebhook: cfg.SlackWebhookURL,
	}

	err = hydra.RegisterWorkflow(worker, quotaChecks)
	if err != nil {
		return fmt.Errorf("unable to register workflow: %w", err)
	}

	err = engine.RegisterCron("* * * * *", "run-quota-checks", func(ctx context.Context, p hydra.CronPayload) error {

		year, month, _ := time.UnixMilli(p.ScheduledAt).Date()

		id, startErr := quotaChecks.Start(ctx, engine, year, int(month))
		if startErr != nil {
			return startErr
		}
		logger.Info("enqueued quota check", "id", id)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to register cron: %w", err)
	}

	// Start the worker
	err = worker.Start(ctx)
	if err != nil {
		return fmt.Errorf("unable to start hydra worker: %w", err)
	}

	// Register worker shutdown
	shutdowns.RegisterCtx(func(ctx context.Context) error {
		return worker.Shutdown(ctx)
	})

	logger.Info("controlplane service started successfully")

	return gracefulShutdown(ctx, logger, shutdowns)
}

func gracefulShutdown(ctx context.Context, logger logging.Logger, shutdowns *shutdown.Shutdowns) error {
	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	// Create a channel that closes when the context is done
	done := ctx.Done()

	// Wait for either a signal or context cancellation
	select {
	case <-cShutdown:
		logger.Info("shutting down due to signal")
	case <-done:
		logger.Info("shutting down due to context cancellation")
	}

	// Create a timeout context for the shutdown process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	errs := shutdowns.Shutdown(shutdownCtx)

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred during shutdown: %v", errs)
	}

	logger.Info("controlplane service shut down successfully")
	return nil
}
