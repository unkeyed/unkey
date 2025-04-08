package api

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/routes"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

// nolint:gocognit
func Run(ctx context.Context, cfg Config) error {

	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	clk := clock.New()

	shutdowns := shutdown.New()

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "api",
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

	if cfg.TestMode {
		logger = logger.With("testmode", true)
		logger.Warn("TESTMODE IS ENABLED. This is not secure in production!")
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

	db, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	defer db.Close()

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}
		go func() {
			promListenErr := prom.Listen(ctx, fmt.Sprintf(":%d", cfg.PrometheusPort))
			if promListenErr != nil {
				panic(promListenErr)
			}
		}()
	}

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

	caches, err := caches.New(caches.Config{
		Logger: logger,
		Clock:  clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	srv, err := zen.New(zen.Config{
		Logger: logger,
		Flags: &zen.Flags{
			TestMode: cfg.TestMode,
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)

	}
	shutdowns.Register(srv.Shutdown)

	validator, err := validation.New()
	if err != nil {
		return fmt.Errorf("unable to create validator: %w", err)
	}

	keySvc, err := keys.New(keys.Config{
		Logger:   logger,
		DB:       db,
		Clock:    clk,
		KeyCache: caches.KeyByHash,
	})
	if err != nil {
		return fmt.Errorf("unable to create key service: %w", err)
	}

	ctr, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: cfg.RedisUrl,
		Logger:   logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create counter: %w", err)
	}

	rlSvc, err := ratelimit.New(ratelimit.Config{
		Logger:  logger,
		Clock:   clk,
		Counter: ctr,
	})
	if err != nil {
		return fmt.Errorf("unable to create ratelimit service: %w", err)
	}

	p, err := permissions.New(permissions.Config{
		DB:     db,
		Logger: logger,
		Clock:  clk,
		Cache:  caches.PermissionsByKeyId,
	})
	if err != nil {
		return fmt.Errorf("unable to create permissions service: %w", err)
	}

	routes.Register(srv, &routes.Services{
		Logger:      logger,
		Database:    db,
		ClickHouse:  ch,
		Keys:        keySvc,
		Validator:   validator,
		Ratelimit:   rlSvc,
		Permissions: p,
		Caches:      caches,
	})

	go func() {
		listenErr := srv.Listen(ctx, fmt.Sprintf(":%d", cfg.HttpPort))
		if listenErr != nil {
			panic(listenErr)
		}
	}()

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
	return nil
}
