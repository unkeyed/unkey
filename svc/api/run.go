package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/cluster"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/routes"
)

// nolint:gocognit
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger.SetSampler(logger.TailSampler{
		SlowThreshold: cfg.LogSlowThreshold,
		SampleRate:    cfg.LogSampleRate,
	})
	logger.AddBaseAttrs(slog.GroupAttrs("instance",
		slog.String("id", cfg.InstanceID),
		slog.String("platform", cfg.Platform),
		slog.String("region", cfg.Region),
		slog.String("version", version.Version),
	))

	if cfg.TestMode {
		logger.AddBaseAttrs(slog.Bool("testmode", true))
	}
	if cfg.TLSConfig != nil {
		logger.AddBaseAttrs(slog.Bool("tls_enabled", true))
	}

	clk := clock.New()

	// This is a little ugly, but the best we can do to resolve the circular dependency until we rework the logger.
	var shutdownGrafana func(context.Context) error
	if cfg.OtelEnabled {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:     "api",
			Version:         version.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	db, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	r.Defer(db.Close)

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

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickhouseURL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL: cfg.ClickhouseURL,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
	}

	// Caches will be created after invalidation consumer is set up
	srv, err := zen.New(zen.Config{
		Flags: &zen.Flags{
			TestMode: cfg.TestMode,
		},
		TLS:                cfg.TLSConfig,
		EnableH2C:          false,
		MaxRequestBodySize: cfg.MaxRequestBodySize,
		ReadTimeout:        0,
		WriteTimeout:       0,
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)
	}
	r.RegisterHealth(srv.Mux())

	r.DeferCtx(srv.Shutdown)

	validator, err := validation.New()
	if err != nil {
		return fmt.Errorf("unable to create validator: %w", err)
	}

	ctr, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: cfg.RedisUrl,
	})
	if err != nil {
		return fmt.Errorf("unable to create counter: %w", err)
	}

	rlSvc, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: ctr,
	})
	if err != nil {
		return fmt.Errorf("unable to create ratelimit service: %w", err)
	}

	// Key service will be created after caches
	r.Defer(rlSvc.Close)

	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		DB:      db,
		Counter: ctr,
		TTL:     60 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("unable to create usage limiter service: %w", err)
	}

	var vaultClient vault.VaultServiceClient
	if cfg.VaultURL != "" {
		vaultClient = vault.NewConnectVaultServiceClient(
			vaultv1connect.NewVaultServiceClient(
				&http.Client{},
				cfg.VaultURL,
				connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", cfg.VaultToken),
				})),
			),
		)
	}

	auditlogSvc, err := auditlogs.New(auditlogs.Config{
		DB: db,
	})
	if err != nil {
		return fmt.Errorf("unable to create auditlogs service: %w", err)
	}

	// Initialize gossip-based cache invalidation
	var broadcaster clustering.Broadcaster
	if cfg.GossipEnabled {
		logger.Info("Initializing gossip cluster for cache invalidation",
			"region", cfg.Region,
			"instanceID", cfg.InstanceID,
		)

		gossipBroadcaster := clustering.NewGossipBroadcaster()

		mux := cluster.NewMessageMux()
		mux.HandleCacheInvalidation(gossipBroadcaster.OnMessage)

		lanSeeds := cluster.ResolveDNSSeeds(cfg.GossipLANSeeds, cfg.GossipLANPort)
		wanSeeds := cluster.ResolveDNSSeeds(cfg.GossipWANSeeds, cfg.GossipWANPort)

		gossipCluster, clusterErr := cluster.New(cluster.Config{
			Region:      cfg.Region,
			NodeID:      cfg.InstanceID,
			BindAddr:    cfg.GossipBindAddr,
			BindPort:    cfg.GossipLANPort,
			WANBindPort: cfg.GossipWANPort,
			LANSeeds:    lanSeeds,
			WANSeeds:    wanSeeds,
			OnMessage:   mux.OnMessage,
		})
		if clusterErr != nil {
			return fmt.Errorf("unable to create gossip cluster: %w", clusterErr)
		}

		gossipBroadcaster.SetCluster(gossipCluster)
		broadcaster = gossipBroadcaster
		r.Defer(gossipCluster.Close)
	}

	caches, err := caches.New(caches.Config{
		Clock:       clk,
		Broadcaster: broadcaster,
		NodeID:      cfg.InstanceID,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	keySvc, err := keys.New(keys.Config{
		DB:           db,
		KeyCache:     caches.VerificationKeyByHash,
		RateLimiter:  rlSvc,
		RBAC:         rbac.New(),
		Clickhouse:   ch,
		Region:       cfg.Region,
		UsageLimiter: ulSvc,
	})
	if err != nil {
		return fmt.Errorf("unable to create key service: %w", err)
	}

	r.Defer(keySvc.Close)
	r.Defer(ctr.Close)

	// Initialize analytics connection manager
	analyticsConnMgr := analytics.NewNoopConnectionManager()
	if cfg.ClickhouseAnalyticsURL != "" && vaultClient != nil {
		analyticsConnMgr, err = analytics.NewConnectionManager(analytics.ConnectionManagerConfig{
			SettingsCache: caches.ClickhouseSetting,
			Database:      db,
			Clock:         clk,
			BaseURL:       cfg.ClickhouseAnalyticsURL,
			Vault:         vaultClient,
		})
		if err != nil {
			return fmt.Errorf("unable to create analytics connection manager: %w", err)
		}
	}

	// Initialize CTRL deployment client
	ctrlDeploymentClient := ctrl.NewConnectDeployServiceClient(
		ctrlv1connect.NewDeployServiceClient(
			&http.Client{},
			cfg.CtrlURL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", cfg.CtrlToken),
			})),
		),
	)

	logger.Info("CTRL clients initialized", "url", cfg.CtrlURL)

	routes.Register(srv, &routes.Services{
		Database:                   db,
		ClickHouse:                 ch,
		Keys:                       keySvc,
		Validator:                  validator,
		Ratelimit:                  rlSvc,
		Auditlogs:                  auditlogSvc,
		Caches:                     caches,
		Vault:                      vaultClient,
		ChproxyToken:               cfg.ChproxyToken,
		CtrlDeploymentClient:       ctrlDeploymentClient,
		PprofEnabled:               cfg.PprofEnabled,
		PprofUsername:              cfg.PprofUsername,
		PprofPassword:              cfg.PprofPassword,
		UsageLimiter:               ulSvc,
		AnalyticsConnectionManager: analyticsConnMgr,
	},
		zen.InstanceInfo{
			ID:     cfg.InstanceID,
			Region: cfg.Region,
		})

	if cfg.Listener == nil {
		// Create listener from HttpPort (production)
		cfg.Listener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
		if err != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.HttpPort, err)
		}
	}

	r.Go(func(ctx context.Context) error {
		serveErr := srv.Serve(ctx, cfg.Listener)
		if serveErr != nil && !errors.Is(serveErr, context.Canceled) && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("server failed: %w", serveErr)
		}
		logger.Info("API server started successfully")
		return nil
	})

	// Wait for either OS signals or context cancellation, then shutdown
	if err := r.Wait(ctx, runner.WithTimeout(time.Minute)); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("API server shut down successfully")
	return nil
}
