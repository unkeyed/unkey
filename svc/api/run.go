package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"connectrpc.com/connect"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
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
	debugpkg "github.com/unkeyed/unkey/pkg/debug"
	"github.com/unkeyed/unkey/pkg/eventstream"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/shutdown"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/vault/storage"
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

	if cfg.TLSConfig != nil {
		logger.Info("TLS is enabled, server will use HTTPS")
	}

	// Enable debug cache headers if configured
	if cfg.DebugCacheHeaders {
		debugpkg.EnableCacheHeaders()
		logger.Info("Debug cache headers enabled - X-Unkey-Debug-Cache headers will be added to responses")
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

	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close db", "error", err)
		}
	}()

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
			serveErr := prom.Serve(ctx, promListener)
			if serveErr != nil {
				panic(serveErr)
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

	// Caches will be created after invalidation consumer is set up
	srv, err := zen.New(zen.Config{
		Logger: logger,
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

	shutdowns.RegisterCtx(srv.Shutdown)

	validator, err := validation.New()
	if err != nil {
		return fmt.Errorf("unable to create validator: %w", err)
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

	// Key service will be created after caches
	shutdowns.Register(rlSvc.Close)

	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		Logger:  logger,
		DB:      db,
		Counter: ctr,
		TTL:     60 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("unable to create usage limiter service: %w", err)
	}

	var vaultSvc *vault.Service
	if len(cfg.VaultMasterKeys) > 0 && cfg.VaultS3 != nil {
		var vaultStorage storage.Storage
		vaultStorage, err = storage.NewS3(storage.S3Config{
			Logger:            logger,
			S3URL:             cfg.VaultS3.URL,
			S3Bucket:          cfg.VaultS3.Bucket,
			S3AccessKeyID:     cfg.VaultS3.AccessKeyID,
			S3AccessKeySecret: cfg.VaultS3.AccessKeySecret,
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

	auditlogSvc, err := auditlogs.New(auditlogs.Config{
		Logger: logger,
		DB:     db,
	})
	if err != nil {
		return fmt.Errorf("unable to create auditlogs service: %w", err)
	}

	// Initialize cache invalidation topic
	cacheInvalidationTopic := eventstream.NewNoopTopic[*cachev1.CacheInvalidationEvent]()
	if len(cfg.KafkaBrokers) > 0 {
		logger.Info("Initializing cache invalidation topic", "brokers", cfg.KafkaBrokers, "instanceID", cfg.InstanceID)

		topicName := cfg.CacheInvalidationTopic
		if topicName == "" {
			topicName = DefaultCacheInvalidationTopic
		}

		cacheInvalidationTopic, err = eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
			Brokers:    cfg.KafkaBrokers,
			Topic:      topicName,
			InstanceID: cfg.InstanceID,
			Logger:     logger,
		})
		if err != nil {
			return fmt.Errorf("unable to create cache invalidation topic: %w", err)
		}

		// Register topic for graceful shutdown
		shutdowns.Register(cacheInvalidationTopic.Close)
	}

	caches, err := caches.New(caches.Config{
		Logger:                 logger,
		Clock:                  clk,
		CacheInvalidationTopic: cacheInvalidationTopic,
		NodeID:                 cfg.InstanceID,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	keySvc, err := keys.New(keys.Config{
		Logger:       logger,
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

	shutdowns.Register(keySvc.Close)
	shutdowns.Register(ctr.Close)

	// Initialize analytics connection manager
	analyticsConnMgr := analytics.NewNoopConnectionManager()
	if cfg.ClickhouseAnalyticsURL != "" && vaultSvc != nil {
		analyticsConnMgr, err = analytics.NewConnectionManager(analytics.ConnectionManagerConfig{
			SettingsCache: caches.ClickhouseSetting,
			Database:      db,
			Logger:        logger,
			Clock:         clk,
			BaseURL:       cfg.ClickhouseAnalyticsURL,
			Vault:         vaultSvc,
		})
		if err != nil {
			return fmt.Errorf("unable to create analytics connection manager: %w", err)
		}
	}

	// Initialize CTRL deployment client using bufconnect
	ctrlDeploymentClient := ctrlv1connect.NewDeploymentServiceClient(
		&http.Client{},
		cfg.CtrlURL,
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", cfg.CtrlToken),
		})),
	)

	logger.Info("CTRL clients initialized", "url", cfg.CtrlURL)

	routes.Register(srv, &routes.Services{
		Logger:                     logger,
		Database:                   db,
		ClickHouse:                 ch,
		Keys:                       keySvc,
		Validator:                  validator,
		Ratelimit:                  rlSvc,
		Auditlogs:                  auditlogSvc,
		Caches:                     caches,
		Vault:                      vaultSvc,
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

	go func() {
		serveErr := srv.Serve(ctx, cfg.Listener)
		if serveErr != nil {
			panic(serveErr)
		}

		logger.Info("API server started successfully")
	}()

	// Wait for either OS signals or context cancellation, then shutdown
	if err := shutdowns.WaitForSignal(ctx, time.Minute); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("API server shut down successfully")
	return nil
}
