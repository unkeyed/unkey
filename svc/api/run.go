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
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/eventstream"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/vault"
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
		SlowThreshold: cfg.Logging.SlowThreshold,
		SampleRate:    cfg.Logging.SampleRate,
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
	if cfg.Otel.Enabled {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:     "api",
			Version:         version.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.Otel.TraceSamplingRate,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	db, err := db.New(db.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
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
	if cfg.ClickHouse.URL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.URL,
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
		RedisURL: cfg.RedisURL,
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

	var vaultClient vault.Client
	if cfg.Vault.URL != "" {
		connectClient := vaultv1connect.NewVaultServiceClient(
			&http.Client{},
			cfg.Vault.URL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", cfg.Vault.Token),
			})),
		)
		vaultClient = vault.NewConnectClient(connectClient)
	}

	auditlogSvc, err := auditlogs.New(auditlogs.Config{
		DB: db,
	})
	if err != nil {
		return fmt.Errorf("unable to create auditlogs service: %w", err)
	}

	// Initialize cache invalidation topic
	cacheInvalidationTopic := eventstream.NewNoopTopic[*cachev1.CacheInvalidationEvent]()
	if cfg.Kafka != nil {
		logger.Info("Initializing cache invalidation topic", "brokers", cfg.Kafka.Brokers, "instanceID", cfg.InstanceID)

		topicName := cfg.Kafka.CacheInvalidationTopic
		if topicName == "" {
			topicName = DefaultCacheInvalidationTopic
		}

		cacheInvalidationTopic, err = eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
			Brokers:    cfg.Kafka.Brokers,
			Topic:      topicName,
			InstanceID: cfg.InstanceID,
		})
		if err != nil {
			return fmt.Errorf("unable to create cache invalidation topic: %w", err)
		}

		// Register topic for graceful shutdown
		r.Defer(cacheInvalidationTopic.Close)
	}

	caches, err := caches.New(caches.Config{
		Clock:                  clk,
		CacheInvalidationTopic: cacheInvalidationTopic,
		NodeID:                 cfg.InstanceID,
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
	if cfg.ClickHouse.AnalyticsURL != "" && vaultClient != nil {
		analyticsConnMgr, err = analytics.NewConnectionManager(analytics.ConnectionManagerConfig{
			SettingsCache: caches.ClickhouseSetting,
			Database:      db,
			Clock:         clk,
			BaseURL:       cfg.ClickHouse.AnalyticsURL,
			Vault:         vaultClient,
		})
		if err != nil {
			return fmt.Errorf("unable to create analytics connection manager: %w", err)
		}
	}

	// Initialize CTRL deployment client using bufconnect
	ctrlDeploymentClient := ctrlv1connect.NewDeployServiceClient(
		&http.Client{},
		cfg.Ctrl.URL,
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", cfg.Ctrl.Token),
		})),
	)

	logger.Info("CTRL clients initialized", "url", cfg.Ctrl.URL)

	var pprofUsername, pprofPassword string
	if cfg.Pprof != nil {
		pprofUsername = cfg.Pprof.Username
		pprofPassword = cfg.Pprof.Password
	}

	routes.Register(srv, &routes.Services{
		Database:                   db,
		ClickHouse:                 ch,
		Keys:                       keySvc,
		Validator:                  validator,
		Ratelimit:                  rlSvc,
		Auditlogs:                  auditlogSvc,
		Caches:                     caches,
		Vault:                      vaultClient,
		ChproxyToken:               cfg.ClickHouse.ProxyToken,
		CtrlDeploymentClient:       ctrlDeploymentClient,
		PprofEnabled:               cfg.Pprof != nil,
		PprofUsername:              pprofUsername,
		PprofPassword:              pprofPassword,
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
