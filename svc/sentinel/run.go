package sentinel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/cluster"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/runner"
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

	logger.SetSampler(logger.TailSampler{
		SlowThreshold: cfg.LogSlowThreshold,
		SampleRate:    cfg.LogSampleRate,
	})

	clk := clock.New()

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
		prom, promErr := prometheus.New()
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		promListener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if listenErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, listenErr)
		}

		r.DeferCtx(prom.Shutdown)
		r.Go(func(ctx context.Context) error {
			serveErr := prom.Serve(ctx, promListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("prometheus server failed: %w", serveErr)
			}
			return nil
		})
	}

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
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

	// Initialize gossip-based cache invalidation
	var broadcaster clustering.Broadcaster
	if cfg.GossipEnabled {
		logger.Info("Initializing gossip cluster for cache invalidation",
			"region", cfg.Region,
			"instanceID", cfg.SentinelID,
		)

		mux := cluster.NewMessageMux()

		lanSeeds := cluster.ResolveDNSSeeds(cfg.GossipLANSeeds, cfg.GossipLANPort)
		wanSeeds := cluster.ResolveDNSSeeds(cfg.GossipWANSeeds, cfg.GossipWANPort)

		gossipCluster, clusterErr := cluster.New(cluster.Config{
			Region:      cfg.Region,
			NodeID:      cfg.SentinelID,
			BindAddr:    cfg.GossipBindAddr,
			BindPort:    cfg.GossipLANPort,
			WANBindPort: cfg.GossipWANPort,
			LANSeeds:    lanSeeds,
			WANSeeds:    wanSeeds,
			SecretKey:   nil, // Sentinel gossip is locked down via CiliumNetworkPolicy
			OnMessage:   mux.OnMessage,
		})
		if clusterErr != nil {
			logger.Error("Failed to create gossip cluster, continuing without cluster cache invalidation",
				"error", clusterErr,
			)
		} else {
			gossipBroadcaster := clustering.NewGossipBroadcaster(gossipCluster)
			cluster.Subscribe(mux, gossipBroadcaster.HandleCacheInvalidation)
			broadcaster = gossipBroadcaster
			r.Defer(gossipCluster.Close)
		}
	}

	routerSvc, err := router.New(router.Config{
		DB:            database,
		Clock:         clk,
		EnvironmentID: cfg.EnvironmentID,
		Region:        cfg.Region,
		Broadcaster:   broadcaster,
		NodeID:        cfg.SentinelID,
	})
	if err != nil {
		return fmt.Errorf("unable to create router service: %w", err)
	}
	r.Defer(routerSvc.Close)

	svcs := &routes.Services{
		RouterService:      routerSvc,
		Clock:              clk,
		WorkspaceID:        cfg.WorkspaceID,
		EnvironmentID:      cfg.EnvironmentID,
		SentinelID:         cfg.SentinelID,
		Region:             cfg.Region,
		ClickHouse:         ch,
		MaxRequestBodySize: maxRequestBodySize,
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
