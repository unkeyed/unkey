package sentinel

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/shutdown"
	"github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/routes"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

// maxRequestBodySize This will be moved to cfg in a later PR.
const maxRequestBodySize = 1024 * 1024 // 1MB limit for logging request bodies
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	shutdowns := shutdown.New()

	// Initialize OTEL before creating logger so the logger picks up the OTLP handler
	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(
			ctx,
			otel.Config{
				Application:     "sentinel",
				Version:         version.Version,
				InstanceID:      cfg.SentinelID,
				CloudRegion:     cfg.Region,
				TraceSampleRate: cfg.OtelTraceSamplingRate,
			},
			shutdowns,
		)

		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

	// Create logger after InitGrafana so it picks up the OTLP handler
	logger := logging.New()
	if cfg.SentinelID != "" {
		logger = logger.With(slog.String("sentinelID", cfg.SentinelID))
	}

	logger = logger.With(
		slog.String("workspaceID", cfg.WorkspaceID),
		slog.String("environmentID", cfg.EnvironmentID),
	)

	if cfg.Region != "" {
		logger = logger.With(slog.String("region", cfg.Region))
	}

	if version.Version != "" {
		logger = logger.With(slog.String("version", version.Version))
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	clk := clock.New()

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		go func() {
			promListener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
			if listenErr != nil {
				panic(listenErr)
			}
			if serveErr := prom.Serve(ctx, promListener); serveErr != nil {
				panic(serveErr)
			}
		}()
	}

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}
	shutdowns.Register(database.Close)

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickhouseURL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL:    cfg.ClickhouseURL,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
		shutdowns.Register(ch.Close)
	}

	routerSvc, err := router.New(router.Config{
		Logger:        logger,
		DB:            database,
		Clock:         clk,
		EnvironmentID: cfg.EnvironmentID,
		Region:        cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("unable to create router service: %w", err)
	}

	svcs := &routes.Services{
		Logger:                 logger,
		RouterService:          routerSvc,
		Clock:                  clk,
		WorkspaceID:            cfg.WorkspaceID,
		EnvironmentID:          cfg.EnvironmentID,
		SentinelID:             cfg.SentinelID,
		Region:                 cfg.Region,
		ClickHouse:             ch,
		MaxRequestBodySize:     maxRequestBodySize,
		WideSuccessSampleRate: cfg.WideSuccessSampleRate,
		WideSlowThresholdMs:   cfg.WideSlowThresholdMs,
		Image:                  "", // TODO: add Image to sentinel config
	}

	srv, err := zen.New(zen.Config{
		Logger:             logger,
		TLS:                nil,
		Flags:              nil,
		EnableH2C:          true,
		MaxRequestBodySize: maxRequestBodySize,
		ReadTimeout:        0,
		WriteTimeout:       0,
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)
	}
	shutdowns.RegisterCtx(srv.Shutdown)

	routes.Register(srv, svcs)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
	if err != nil {
		return fmt.Errorf("unable to create listener: %w", err)
	}

	go func() {
		logger.Info("Sentinel server started", "addr", listener.Addr().String())
		if serveErr := srv.Serve(ctx, listener); serveErr != nil {
			logger.Error("Server error", "error", serveErr)
		}
	}()

	if err := shutdowns.WaitForSignal(ctx, 0); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Sentinel server shut down successfully")
	return nil
}
