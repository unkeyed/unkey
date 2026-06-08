package api

import (
	"context"
	"database/sql"
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
	"github.com/unkeyed/unkey/internal/services/portal"
	"github.com/unkeyed/unkey/internal/services/ratelimit"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/auth"
	authjwt "github.com/unkeyed/unkey/pkg/auth/jwt"
	portalsession "github.com/unkeyed/unkey/pkg/auth/portal_session"
	rootkey "github.com/unkeyed/unkey/pkg/auth/root_key"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/pkg/uid"
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

	if cfg.TLS.CertFile != "" {
		tlsCfg, tlsErr := tls.NewFromFiles(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if tlsErr != nil {
			return fmt.Errorf("unable to load TLS config: %w", tlsErr)
		}
		cfg.TLSConfig = tlsCfg
	}
	if cfg.InstanceID == "" {
		cfg.InstanceID = uid.New(uid.InstancePrefix)
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
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
		slog.String("version", buildinfo.Version),
	))

	if cfg.Test.Enabled {
		logger.AddBaseAttrs(slog.Bool("testmode", true))
	}
	if cfg.TLSConfig != nil {
		logger.AddBaseAttrs(slog.Bool("tls_enabled", true))
	}

	clk := cfg.Clock

	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	lazy.SetRegistry(reg)
	buildinfo.RegisterBuildInfoMetrics("api")

	// This is a little ugly, but the best we can do to resolve the circular dependency until we rework the logger.
	var shutdownGrafana func(context.Context) error
	if cfg.Observability.Tracing != nil {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:        "api",
			InstanceID:         cfg.InstanceID,
			CloudRegion:        cfg.Region,
			TraceSampleRate:    cfg.Observability.Tracing.SampleRate,
			PrometheusGatherer: reg,
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
		prom, promErr := prometheus.NewWithRegistry(reg)
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
			TestMode: cfg.Test.Enabled,
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

	var ctr counter.Counter
	if cfg.Test.Counter != nil {
		ctr = cfg.Test.Counter
	} else {
		ctr, err = counter.NewRedis(counter.RedisConfig{
			RedisURL: cfg.RedisURL,
		})
		if err != nil {
			return fmt.Errorf("unable to create counter: %w", err)
		}
	}

	rlSvc, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: ctr,
		DB:      db.ToMySQL(database),
		Region:  cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("unable to create ratelimit service: %w", err)
	}

	// Key service will be created after caches
	r.Defer(rlSvc.Close)

	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		FindKeyCredits: func(ctx context.Context, keyID string) (int64, bool, error) {
			limit, err := db.WithRetryContext(ctx, func() (sql.NullInt64, error) {
				return db.Query.FindKeyCredits(ctx, database.RO(), keyID)
			})
			if err != nil {
				return 0, false, err
			}
			return limit.Int64, limit.Valid, nil
		},
		DecrementKeyCredits: func(ctx context.Context, keyID string, cost int64) error {
			return db.Query.UpdateKeyCreditsDecrement(ctx, database.RW(), db.UpdateKeyCreditsDecrementParams{
				ID:      keyID,
				Credits: sql.NullInt64{Int64: cost, Valid: true},
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

	caches, err := caches.New(caches.Config{
		Clock:  clk,
		NodeID: cfg.InstanceID,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	keySvc, err := keys.New(keys.Config{
		DB:           db.ToMySQL(database),
		KeyCache:     caches.VerificationKeyByHash,
		RateLimiter:  rlSvc,
		RBAC:         rbac.New(),
		Region:       cfg.Region,
		UsageLimiter: ulSvc,
	})
	if err != nil {
		return fmt.Errorf("unable to create key service: %w", err)
	}
	portalSvc := portal.New(portal.Config{
		DB:           database,
		SessionCache: caches.PortalSession,
	})
	jwtWorkspaceLookup := authjwt.WorkspaceLookupFunc(func(ctx context.Context, orgID string) (string, error) {
		workspace, lookupErr := db.WithRetryContext(ctx, func() (db.Workspace, error) {
			return db.Query.FindWorkspaceByOrgID(ctx, database.RO(), orgID)
		})
		if lookupErr != nil {
			return "", lookupErr
		}
		return workspace.ID, nil
	})

	authResolvers := []auth.Resolver{}
	for i, authConfig := range cfg.Auth {
		switch authConfig := authConfig.(type) {
		case JWTAuthConfig:
			jwtSecrets := make([][]byte, 0, len(authConfig.Secrets))
			for j, secret := range authConfig.Secrets {
				if secret == "" {
					return fmt.Errorf("auth[%d].secrets[%d] must not be empty", i, j)
				}
				jwtSecrets = append(jwtSecrets, []byte(secret))
			}
			if len(jwtSecrets) > 0 {
				jwtResolver, jwtErr := authjwt.NewResolver(jwtWorkspaceLookup, authConfig.Issuer, authConfig.Audience, jwtSecrets...)
				if jwtErr != nil {
					return fmt.Errorf("unable to create JWT auth resolver from auth[%d]: %w", i, jwtErr)
				}
				authResolvers = append(authResolvers, jwtResolver)
				continue
			}
			if authConfig.JWKSURL != "" {
				jwtResolver, jwtErr := authjwt.NewResolverWithJWKSURL(ctx, jwtWorkspaceLookup, authConfig.Issuer, authConfig.Audience, authConfig.JWKSURL)
				if jwtErr != nil {
					return fmt.Errorf("unable to create JWT auth resolver from auth[%d]: %w", i, jwtErr)
				}
				authResolvers = append(authResolvers, jwtResolver)
				continue
			}
			return fmt.Errorf("auth[%d] jwt requires exactly one of secrets or jwks_url", i)
		case PortalSessionAuthConfig:
			authResolvers = append(authResolvers, portalsession.NewResolver(portalSvc))
		case RootKeyAuthConfig:
			authResolvers = append(authResolvers, rootkey.NewResolver(keySvc))
		default:
			return fmt.Errorf("unsupported auth config at auth[%d]", i)
		}
	}
	authSvc := auth.New(authResolvers...)

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
		KeyVerifications:     keyVerifications,
		Keys:                 keySvc,
		Auth:                 authSvc,
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
		PortalBaseURL:              cfg.PortalBaseURL,
	},
		zen.InstanceInfo{
			ID:     cfg.InstanceID,
			Region: cfg.Region,
		})

	listener := cfg.Test.Listener
	if listener == nil {
		// Create listener from HttpPort (production)
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
		if err != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.HttpPort, err)
		}
	}

	r.Go(func(ctx context.Context) error {
		serveErr := srv.Serve(ctx, listener)
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
