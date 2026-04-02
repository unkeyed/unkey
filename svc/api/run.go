package api

import (
	"context"
	"database/sql"
	"encoding/base64"
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
	"github.com/unkeyed/unkey/internal/services/sessionauth"
	"github.com/unkeyed/unkey/pkg/jwks"

	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
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

	if cfg.Observability.Logging != nil {
		logger.SetSampler(logger.TailSampler{
			SlowThreshold: cfg.Observability.Logging.SlowThreshold,
			SampleRate:    cfg.Observability.Logging.SampleRate,
		})
	}
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
	if cfg.Observability.Tracing != nil {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:     "api",
			Version:         version.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.Observability.Tracing.SampleRate,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	r.Defer(database.Close)

	if cfg.Observability.Metrics != nil {
		prom, promErr := prometheus.New()
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		promListener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Observability.Metrics.PrometheusPort))
		if listenErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.Observability.Metrics.PrometheusPort, listenErr)
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
	apiRequests := batch.NewNoop[schema.ApiRequest]()
	keyVerifications := batch.NewNoop[schema.KeyVerification]()
	ratelimits := batch.NewNoop[schema.Ratelimit]()

	if cfg.ClickHouse.URL != "" {
		chClient, chErr := clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.URL,
		})
		if chErr != nil {
			return fmt.Errorf("unable to create clickhouse: %w", chErr)
		}
		ch = chClient

		apiRequests = clickhouse.NewBuffer[schema.ApiRequest](chClient, "default.api_requests_raw_v2", clickhouse.BufferConfig{
			Name:          "api_requests",
			BatchSize:     10_000,
			BufferSize:    20_000,
			FlushInterval: 5 * time.Second,
			Consumers:     2,
			Drop:          true,
			OnFlushError:  nil,
		})
		keyVerifications = clickhouse.NewBuffer[schema.KeyVerification](chClient, "default.key_verifications_raw_v2", clickhouse.BufferConfig{
			Name:          "key_verifications",
			BatchSize:     10_000,
			BufferSize:    20_000,
			FlushInterval: 5 * time.Second,
			Consumers:     2,
			Drop:          true,
			OnFlushError:  nil,
		})
		ratelimits = clickhouse.NewBuffer[schema.Ratelimit](chClient, "default.ratelimits_raw_v2", clickhouse.BufferConfig{
			Name:          "ratelimits",
			BatchSize:     10_000,
			BufferSize:    20_000,
			FlushInterval: 5 * time.Second,
			Consumers:     2,
			Drop:          true,
			OnFlushError:  nil,
		})

		// Close buffers before connection (LIFO)
		r.Defer(func() error { apiRequests.Close(); return nil })
		r.Defer(func() error { keyVerifications.Close(); return nil })
		r.Defer(func() error { ratelimits.Close(); return nil })
		r.Defer(chClient.Close)
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
		FindKeyCredits: func(ctx context.Context, keyID string) (int32, bool, error) {
			limit, err := db.WithRetryContext(ctx, func() (sql.NullInt32, error) {
				return db.Query.FindKeyCredits(ctx, database.RO(), keyID)
			})
			if err != nil {
				return 0, false, err
			}
			return limit.Int32, limit.Valid, nil
		},
		DecrementKeyCredits: func(ctx context.Context, keyID string, cost int32) error {
			return db.Query.UpdateKeyCreditsDecrement(ctx, database.RW(), db.UpdateKeyCreditsDecrementParams{
				ID:      keyID,
				Credits: sql.NullInt32{Int32: cost, Valid: true},
			})
		},
		Counter: ctr,
		TTL:     60 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("unable to create usage limiter service: %w", err)
	}

	var vaultClient vault.VaultServiceClient
	if cfg.Vault.URL != "" {
		vaultClient = vault.NewConnectVaultServiceClient(vaultv1connect.NewVaultServiceClient(
			&http.Client{},
			cfg.Vault.URL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", cfg.Vault.Token),
			})),
		))
	}

	auditlogSvc, err := auditlogs.New(auditlogs.Config{
		DB: database,
	})
	if err != nil {
		return fmt.Errorf("unable to create auditlogs service: %w", err)
	}

	// Initialize gossip-based cache invalidation
	var broadcaster clustering.Broadcaster
	if cfg.Gossip != nil {
		logger.Info("Initializing gossip cluster for cache invalidation",
			"region", cfg.Region,
			"instanceID", cfg.InstanceID,
		)

		mux := cluster.NewMessageMux()

		lanSeeds := cluster.ResolveDNSSeeds(cfg.Gossip.LANSeeds, cfg.Gossip.LANPort)
		wanSeeds := cluster.ResolveDNSSeeds(cfg.Gossip.WANSeeds, cfg.Gossip.WANPort)

		var secretKey []byte
		if cfg.Gossip.SecretKey != "" {
			var decodeErr error
			secretKey, decodeErr = base64.StdEncoding.DecodeString(cfg.Gossip.SecretKey)
			if decodeErr != nil {
				return fmt.Errorf("unable to decode gossip secret key: %w", decodeErr)
			}
		}

		gossipCluster, clusterErr := cluster.New(cluster.Config{
			Region:           cfg.Region,
			NodeID:           cfg.InstanceID,
			BindAddr:         cfg.Gossip.BindAddr,
			BindPort:         cfg.Gossip.LANPort,
			WANBindPort:      cfg.Gossip.WANPort,
			WANAdvertiseAddr: cfg.Gossip.WANAdvertiseAddr,
			LANSeeds:         lanSeeds,
			WANSeeds:         wanSeeds,
			SecretKey:        secretKey,
			OnMessage:        mux.OnMessage,
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

	caches, err := caches.New(caches.Config{
		Clock:       clk,
		Broadcaster: broadcaster,
		NodeID:      cfg.InstanceID,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	keySvc, err := keys.New(keys.Config{
		DB:               database,
		KeyCache:         caches.VerificationKeyByHash,
		QuotaCache:       caches.WorkspaceQuota,
		RateLimiter:      rlSvc,
		RBAC:             rbac.New(),
		KeyVerifications: keyVerifications,
		Region:           cfg.Region,
		UsageLimiter:     ulSvc,
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
			Database:      database,
			Clock:         clk,
			BaseURL:       cfg.ClickHouse.AnalyticsURL,
			Vault:         vaultClient,
		})
		if err != nil {
			return fmt.Errorf("unable to create analytics connection manager: %w", err)
		}
	}

	// Initialize session auth service
	var sessionAuthSvc sessionauth.Service
	if cfg.SessionAuth != nil {
		switch cfg.SessionAuth.Provider {
		case "jwks":
			keySet := jwks.NewRemoteKeySet(cfg.SessionAuth.JWKSURL)
			sessionAuthSvc = sessionauth.NewJWKS(sessionauth.JWKSConfig{
				KeySet: keySet,
				Issuer: cfg.SessionAuth.Issuer,
				DB:     database,
			})
			logger.Info("Session auth initialized with JWKS provider",
				"jwks_url", cfg.SessionAuth.JWKSURL,
				"issuer", cfg.SessionAuth.Issuer,
			)
		case "local":
			sessionAuthSvc = sessionauth.NewLocal(cfg.SessionAuth.LocalWorkspaceID)
			logger.Info("Session auth initialized with local provider",
				"workspace_id", cfg.SessionAuth.LocalWorkspaceID,
			)
		default:
			return fmt.Errorf("unknown session auth provider: %q", cfg.SessionAuth.Provider)
		}
	}

	// Initialize control plane deployment client
	ctrlDeploymentClient := ctrl.NewConnectDeployServiceClient(
		ctrlv1connect.NewDeployServiceClient(
			&http.Client{},
			cfg.Control.URL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", cfg.Control.Token),
			})),
		),
	)

	logger.Info("Control plane clients initialized", "url", cfg.Control.URL)

	pprofEnabled := cfg.Pprof != nil && cfg.Pprof.Username != "" && cfg.Pprof.Password != ""
	var pprofUsername, pprofPassword string
	if pprofEnabled {
		pprofUsername = cfg.Pprof.Username
		pprofPassword = cfg.Pprof.Password
	}

	routes.Register(srv, &routes.Services{
		Database:             database,
		ClickHouse:           ch,
		ApiRequests:          apiRequests,
		RatelimitEvents:      ratelimits,
		Keys:                 keySvc,
		Validator:            validator,
		Ratelimit:            rlSvc,
		Auditlogs:            auditlogSvc,
		Caches:               caches,
		Vault:                vaultClient,
		CtrlDeploymentClient: ctrlDeploymentClient,
		PprofEnabled:         pprofEnabled,
		PprofUsername:        pprofUsername,
		PprofPassword:        pprofPassword,

		UsageLimiter:               ulSvc,
		AnalyticsConnectionManager: analyticsConnMgr,
		SessionAuth:                sessionAuthSvc,
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
