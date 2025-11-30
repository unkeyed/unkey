package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"

	"github.com/unkeyed/unkey/go/apps/gateway/routes"
	"github.com/unkeyed/unkey/go/apps/gateway/services/router"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New()
	if cfg.GatewayID != "" {
		logger = logger.With(slog.String("gatewayID", cfg.GatewayID))
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
		Logger:        logger,
		RouterService: routerSvc,
		Clock:         clk,
		EnvironmentID: cfg.EnvironmentID,
		Region:        cfg.Region,
	}

	srv, err := zen.New(zen.Config{
		Logger: logger,
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
		logger.Info("Gateway server started", "addr", listener.Addr().String())
		if serveErr := srv.Serve(ctx, listener); serveErr != nil {
			logger.Error("Server error", "error", serveErr)
		}
	}()

	if err := shutdowns.WaitForSignal(ctx, 0); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Gateway server shut down successfully")
	return nil
}
