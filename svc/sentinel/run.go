package sentinel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/o11y"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/middleware"
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

	logger.SetSampler(logger.TailSampler{
		SlowThreshold: cfg.LogSlowThreshold,
		SampleRate:    cfg.LogSampleRate,
	})

	clk := clock.New()

	obs, err := o11y.New(ctx, o11y.Config{
		Service:    "sentinel",
		Region:     cfg.Region,
		Version:    version.Version,
		InstanceID: cfg.SentinelID,
	})
	if err != nil {
		return fmt.Errorf("unable to create o11y: %w", err)
	}
	snMetrics := middleware.NewPrometheusMetrics(obs.Registry())

	// Initialize OTEL before creating logger so the logger picks up the OTLP handler
	var shutdownGrafana func(context.Context) error
	if cfg.OtelEnabled {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:     "sentinel",
			Version:         version.Version,
			InstanceID:      cfg.SentinelID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	// Add base attributes to global logger
	logger.AddBaseAttrs(slog.GroupAttrs("instance",
		slog.String("sentinelID", cfg.SentinelID),
		slog.String("workspaceID", cfg.WorkspaceID),
		slog.String("environmentID", cfg.EnvironmentID),
		slog.String("region", cfg.Region),
		slog.String("version", version.Version),
	))

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	if cfg.PrometheusPort > 0 {
		promSrv, promErr := zen.New(zen.Config{
			MaxRequestBodySize: 0,
			Flags:              nil,
			TLS:                nil,
			EnableH2C:          false,
			ReadTimeout:        0,
			WriteTimeout:       0,
		})
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}
		metricsHandler := obs.Handler()
		promSrv.RegisterRoute([]zen.Middleware{}, zen.NewRoute("GET", "/metrics", func(ctx context.Context, s *zen.Session) error {
			metricsHandler.ServeHTTP(s.ResponseWriter(), s.Request())
			return nil
		}))

		promListener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if listenErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, listenErr)
		}

		r.DeferCtx(promSrv.Shutdown)
		r.Go(func(ctx context.Context) error {
			serveErr := promSrv.Serve(ctx, promListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("prometheus server failed: %w", serveErr)
			}
			return nil
		})
	}

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Metrics:     obs.Metrics,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}
	r.Defer(database.Close)

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickhouseURL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL: cfg.ClickhouseURL,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
		r.Defer(ch.Close)
	}

	routerSvc, err := router.New(router.Config{
		DB:            database,
		Clock:         clk,
		EnvironmentID: cfg.EnvironmentID,
		Region:        cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("unable to create router service: %w", err)
	}

	svcs := &routes.Services{
		RouterService:      routerSvc,
		Clock:              clk,
		WorkspaceID:        cfg.WorkspaceID,
		EnvironmentID:      cfg.EnvironmentID,
		SentinelID:         cfg.SentinelID,
		Region:             cfg.Region,
		ClickHouse:         ch,
		MaxRequestBodySize: maxRequestBodySize,
		ZenMetrics:         obs.Metrics,
		ObsMetrics:         snMetrics,
	}

	srv, err := zen.New(zen.Config{
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
	r.RegisterHealth(srv.Mux())
	r.DeferCtx(srv.Shutdown)

	routes.Register(srv, svcs)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
	if err != nil {
		return fmt.Errorf("unable to create listener: %w", err)
	}

	r.Go(func(ctx context.Context) error {
		logger.Info("Sentinel server started", "addr", listener.Addr().String())
		if serveErr := srv.Serve(ctx, listener); serveErr != nil && !errors.Is(serveErr, context.Canceled) {
			return fmt.Errorf("server error: %w", serveErr)
		}
		return nil
	})

	if err := r.Wait(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Sentinel server shut down successfully")
	return nil
}
