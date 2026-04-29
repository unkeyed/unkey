package frontline

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
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/keys"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/cluster"
	"github.com/unkeyed/unkey/pkg/counter"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	pprofRoute "github.com/unkeyed/unkey/pkg/pprof"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/caches"
	"github.com/unkeyed/unkey/svc/frontline/internal/certmanager"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/routes"
)

// Run starts the frontline server.
//
// Frontline is the multi-tenant ingress that:
//   - Terminates TLS for customer domains
//   - Resolves the hostname to a deployment + parses its policies
//   - Runs the policy engine (KeyAuth, RateLimit, Firewall)
//   - Forwards directly to a running deployment instance in this region,
//     or hops to a peer frontline in another region when no local instance exists.
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

	clk := clock.New()

	// Initialize OTEL before creating logger so the logger picks up the OTLP handler
	var shutdownGrafana func(context.Context) error
	if cfg.Observability.Tracing != nil {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:        "frontline",
			InstanceID:         cfg.InstanceID,
			CloudRegion:        cfg.Region,
			TraceSampleRate:    cfg.Observability.Tracing.SampleRate,
			PrometheusGatherer: nil,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
		logger.Info("Grafana tracing initialized", "sampleRate", cfg.Observability.Tracing.SampleRate)
	} else {
		logger.Warn("Tracing not configured, skipping Grafana OTEL initialization")
	}

	if cfg.InstanceID != "" {
		logger.AddBaseAttrs(slog.String("instanceID", cfg.InstanceID))
	}

	if cfg.Region != "" {
		logger.AddBaseAttrs(slog.String("region", cfg.Region))
	}

	if buildinfo.Version != "" {
		logger.AddBaseAttrs(slog.String("version", buildinfo.Version))
	}

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	lazy.SetRegistry(reg)
	buildinfo.RegisterBuildInfoMetrics("frontline")

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.NewWithRegistry(reg)
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		promListener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if listenErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, listenErr)
		}

		r.DeferCtx(prom.Shutdown)
		r.Go(func(ctx context.Context) error {
			promListenErr := prom.Serve(ctx, promListener)
			if promListenErr != nil && !errors.Is(promListenErr, context.Canceled) {
				return fmt.Errorf("prometheus server failed: %w", promListenErr)
			}
			return nil
		})
	} else {
		logger.Warn("Prometheus not configured, skipping metrics server")
	}

	if cfg.Pprof != nil {
		pprofPort := cfg.Pprof.Port
		if pprofPort == 0 {
			pprofPort = 6060
		}

		pprofSrv, pprofErr := pprofRoute.New(cfg.Pprof, "/_unkey/internal")
		if pprofErr != nil {
			return fmt.Errorf("unable to create pprof server: %w", pprofErr)
		}
		r.DeferCtx(pprofSrv.Shutdown)

		pprofListener, pprofListenErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", pprofPort))
		if pprofListenErr != nil {
			return fmt.Errorf("unable to listen on 127.0.0.1:%d for pprof: %w", pprofPort, pprofListenErr)
		}

		r.Go(func(ctx context.Context) error {
			logger.Info("Internal pprof server started", "addr", pprofListener.Addr().String())
			serveErr := pprofSrv.Serve(ctx, pprofListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("pprof server error: %w", serveErr)
			}
			return nil
		})
	}

	var vaultClient vault.VaultServiceClient
	if cfg.Vault.URL != "" {
		vaultClient = vault.NewConnectVaultServiceClient(vaultv1connect.NewVaultServiceClient(
			http.DefaultClient,
			cfg.Vault.URL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": "Bearer " + cfg.Vault.Token,
			})),
		))
		logger.Info("Vault client initialized", "url", cfg.Vault.URL)
	} else {
		logger.Warn("Vault not configured, dynamic TLS certificate decryption will be unavailable")
	}

	// internal/db drives the routing/cert lookups against a read-only
	// replica. When the operator omits a dedicated replica DSN we fall back
	// to the primary; pkgdb does the same internally so both pools end up
	// targeting the same endpoint.
	readDSN := cfg.Database.ReadonlyReplica
	if readDSN == "" {
		readDSN = cfg.Database.Primary
	}
	database, databaseClose, err := db.New(readDSN)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	r.Defer(databaseClose)

	// engineDatabase is a separate connection pool used by the policy
	// engine (key verification decrements credits, so write access is
	// required). The frontline routing/cert path keeps using the read
	// replica pool above.
	engineDatabase, err := pkgdb.New(pkgdb.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
	})
	if err != nil {
		return fmt.Errorf("unable to connect to engine database: %w", err)
	}
	r.Defer(engineDatabase.Close)

	frontlineRequests := batch.NewNoop[schema.SentinelRequest]()
	keyVerifications := batch.NewNoop[schema.KeyVerification]()

	var chClient *clickhouse.Client
	if cfg.ClickHouse.URL != "" {
		chClient, err = clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.URL,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}

		frontlineRequests = clickhouse.NewBuffer[schema.SentinelRequest](chClient, "default.sentinel_requests_raw_v1", clickhouse.BufferConfig{
			Name:          "sentinel_requests",
			BatchSize:     cfg.ClickHouse.BatchSize,
			BufferSize:    cfg.ClickHouse.BufferSize,
			FlushInterval: 5 * time.Second,
			Consumers:     cfg.ClickHouse.Consumers,
			Drop:          true,
			OnFlushError:  nil,
		})
		keyVerifications = clickhouse.NewBuffer[schema.KeyVerification](chClient, "default.key_verifications_raw_v2", clickhouse.BufferConfig{
			Name:          "key_verifications",
			BatchSize:     cfg.ClickHouse.BatchSize,
			BufferSize:    cfg.ClickHouse.BufferSize,
			FlushInterval: 5 * time.Second,
			Consumers:     cfg.ClickHouse.Consumers,
			Drop:          true,
			OnFlushError:  nil,
		})

		// Close buffers before the connection (LIFO).
		r.Defer(func() error { frontlineRequests.Close(); return nil })
		r.Defer(func() error { keyVerifications.Close(); return nil })
		r.Defer(chClient.Close)
	}

	var broadcaster clustering.Broadcaster
	if cfg.Gossip != nil {
		logger.Info("Gossip cluster configured, initializing cache invalidation",
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
	} else {
		logger.Warn("Gossip not configured, cache invalidation will be local only")
	}

	cacheSet, err := caches.New(caches.Config{
		Clock:       clk,
		Broadcaster: broadcaster,
		NodeID:      cfg.InstanceID,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}
	r.Defer(cacheSet.Close)

	var certManager certmanager.Service
	if vaultClient != nil {
		certManager = certmanager.New(certmanager.Config{
			DB:                  database,
			TLSCertificateCache: cacheSet.TLSCertificates,
			Vault:               vaultClient,
		})
		logger.Info("Certificate manager initialized with vault-backed decryption")
	} else {
		logger.Warn("Certificate manager not initialized, vault client is nil")
	}

	routerSvc, err := router.New(router.Config{
		Platform:              cfg.Platform,
		Region:                cfg.Region,
		DB:                    database,
		FrontlineRouteCache:   cacheSet.FrontlineRoutes,
		InstancesByDeployment: cacheSet.InstancesByDeployment,
		PolicyCache:           cacheSet.Policies,
	})
	if err != nil {
		return fmt.Errorf("unable to create router service: %w", err)
	}

	upstreamTransports := proxy.NewTransportRegistry()

	// nolint:exhaustruct
	proxySvc, err := proxy.New(proxy.Config{
		InstanceID:         cfg.InstanceID,
		Platform:           cfg.Platform,
		Region:             cfg.Region,
		ApexDomain:         cfg.ApexDomain,
		Clock:              clk,
		MaxHops:            cfg.MaxHops,
		UpstreamTransports: upstreamTransports,
	})
	if err != nil {
		return fmt.Errorf("unable to create proxy service: %w", err)
	}

	policyEngine, err := buildEngine(r, engineDatabase, cfg.Redis.URL, cfg.Region, keyVerifications, clk)
	if err != nil {
		return fmt.Errorf("unable to build policy engine: %w", err)
	}

	tlsConfig, err := buildTlsConfig(cfg, certManager)
	if err != nil {
		return fmt.Errorf("unable to build tls config: %w", err)
	}

	acmeClient := ctrl.NewConnectAcmeServiceClient(ctrlv1connect.NewAcmeServiceClient(ptr.P(http.Client{}), cfg.CtrlAddr))

	svcs := &routes.Services{
		Region:            cfg.Region,
		Platform:          cfg.Platform,
		FrontlineID:       cfg.InstanceID,
		RouterService:     routerSvc,
		ProxyService:      proxySvc,
		Engine:            policyEngine,
		Clock:             clk,
		AcmeClient:        acmeClient,
		ErrorPageRenderer: errorpage.NewRenderer(),
		RequestTimeout:    cfg.RequestTimeout,
		FrontlineRequests: frontlineRequests,
	}

	if cfg.HttpsPort > 0 {
		httpsSrv, httpsErr := zen.New(zen.Config{
			TLS:                tlsConfig,
			ReadTimeout:        -1,
			WriteTimeout:       -1,
			Flags:              nil,
			EnableH2C:          false,
			MaxRequestBodySize: 0,
		})
		if httpsErr != nil {
			return fmt.Errorf("unable to create HTTPS server: %w", httpsErr)
		}
		r.RegisterHealth(httpsSrv.Mux(), "/_unkey/internal/health")
		r.AddReadinessCheck("database", func(ctx context.Context) error {
			return engineDatabase.RW().PingContext(ctx)
		})
		r.DeferCtx(httpsSrv.Shutdown)

		routes.Register(httpsSrv, svcs)

		httpsListener, httpsListenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpsPort))
		if httpsListenErr != nil {
			return fmt.Errorf("unable to create HTTPS listener: %w", httpsListenErr)
		}

		r.Go(func(ctx context.Context) error {
			logger.Info("HTTPS frontline server started",
				"addr", httpsListener.Addr().String(),
				"tlsEnabled", tlsConfig != nil,
			)
			serveErr := httpsSrv.Serve(ctx, httpsListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("https server error: %w", serveErr)
			}
			return nil
		})
	} else {
		logger.Warn("HTTPS server not configured, skipping", "httpsPort", cfg.HttpsPort)
	}

	if cfg.HttpPort > 0 {
		httpSrv, httpErr := zen.New(zen.Config{
			TLS:                nil,
			Flags:              nil,
			EnableH2C:          false,
			MaxRequestBodySize: 0,
			ReadTimeout:        -1,
			WriteTimeout:       -1,
		})
		if httpErr != nil {
			return fmt.Errorf("unable to create HTTP server: %w", httpErr)
		}
		r.RegisterHealth(httpSrv.Mux(), "/_unkey/internal/health")
		r.DeferCtx(httpSrv.Shutdown)

		routes.RegisterHTTPServer(httpSrv, svcs)

		httpListener, httpListenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
		if httpListenErr != nil {
			return fmt.Errorf("unable to create HTTP listener: %w", httpListenErr)
		}

		r.Go(func(ctx context.Context) error {
			logger.Info("HTTP server started", "addr", httpListener.Addr().String())
			serveErr := httpSrv.Serve(ctx, httpListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("http server error: %w", serveErr)
			}
			return nil
		})
	} else {
		logger.Warn("HTTP server not configured, ACME HTTP-01 challenges and HTTP→HTTPS redirects will not work", "httpPort", cfg.HttpPort)
	}

	logger.Info("Frontline server initialized", "region", cfg.Region, "apexDomain", cfg.ApexDomain)

	if err := r.Wait(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Frontline server shut down successfully")
	return nil
}

// buildEngine wires the engine's backing services. Redis is optional — when
// no URL is configured the counter falls back to in-memory. Rate limits in
// that mode are per-replica; distributed enforcement requires Redis.
func buildEngine(
	r *runner.Runner,
	database pkgdb.Database,
	redisURL string,
	region string,
	keyVerifications *batch.BatchProcessor[schema.KeyVerification],
	clk clock.Clock,
) (policies.Evaluator, error) {
	var ctr counter.Counter
	if redisURL != "" {
		redisCtr, err := counter.NewRedis(counter.RedisConfig{
			RedisURL: redisURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create redis counter: %w", err)
		}
		r.Defer(redisCtr.Close)
		ctr = redisCtr
	} else {
		memoryCtr := counter.NewMemory()
		r.Defer(memoryCtr.Close)
		ctr = memoryCtr
	}

	rlSvc, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: ctr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ratelimit service: %w", err)
	}
	r.Defer(rlSvc.Close)

	usageLimiter, err := usagelimiter.NewCounter(usagelimiter.CounterConfig{
		FindKeyCredits: func(ctx context.Context, keyID string) (int64, bool, error) {
			limit, err := pkgdb.WithRetryContext(ctx, func() (sql.NullInt64, error) {
				return pkgdb.Query.FindKeyCredits(ctx, database.RO(), keyID)
			})
			if err != nil {
				return 0, false, err
			}
			return limit.Int64, limit.Valid, nil
		},
		DecrementKeyCredits: func(ctx context.Context, keyID string, cost int64) error {
			return pkgdb.Query.UpdateKeyCreditsDecrement(ctx, database.RW(), pkgdb.UpdateKeyCreditsDecrementParams{
				ID:      keyID,
				Credits: sql.NullInt64{Int64: cost, Valid: true},
			})
		},
		Counter:       ctr,
		TTL:           60 * time.Second,
		ReplayWorkers: 8,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create usage limiter: %w", err)
	}
	r.Defer(usageLimiter.Close)

	keyCache, err := cache.New(cache.Config[string, keysdb.CachedKeyData]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		MaxSize:  100_000,
		Resource: "frontline_key_cache",
		Clock:    clk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create key cache: %w", err)
	}

	keyService, err := keys.New(keys.Config{
		DB:               pkgdb.ToMySQL(database),
		RateLimiter:      rlSvc,
		RBAC:             rbac.New(),
		KeyVerifications: keyVerifications,
		Region:           region,
		UsageLimiter:     usageLimiter,
		KeyCache:         keyCache,
		QuotaCache:       nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create key service: %w", err)
	}

	logger.Info("policy engine initialized")
	eng, err := policies.New(policies.Config{
		KeyService:  keyService,
		RateLimiter: rlSvc,
		Clock:       clk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy engine: %w", err)
	}
	return eng, nil
}
