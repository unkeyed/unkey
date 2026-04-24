package sentinel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/unkeyed/unkey/internal/services/keys"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/quotaretention"
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
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
	"github.com/unkeyed/unkey/svc/sentinel/routes"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

type middlewareEngineCfg struct {
	Runner           *runner.Runner
	RedisURL         string
	Region           string
	Database         db.Database
	KeyVerifications *batch.BatchProcessor[schema.KeyVerification]
	Clock            clock.Clock
}

// maxRequestBodySize This will be moved to cfg in a later PR.
const maxRequestBodySize = 1024 * 1024 // 1MB limit for logging request bodies

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
			Application:        "sentinel",
			InstanceID:         cfg.SentinelID,
			CloudRegion:        cfg.Region,
			TraceSampleRate:    cfg.Observability.Tracing.SampleRate,
			PrometheusGatherer: nil,
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
		slog.String("platform", cfg.Platform),
		slog.String("region", cfg.Region),
		slog.String("version", buildinfo.Version),
	))

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	lazy.SetRegistry(reg)
	buildinfo.RegisterBuildInfoMetrics("sentinel")

	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {
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

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}
	r.Defer(database.Close)

	sentinelRequests := batch.NewNoop[schema.SentinelRequest]()
	keyVerifications := batch.NewNoop[schema.KeyVerification]()

	var chClient *clickhouse.Client
	if cfg.ClickHouse.URL != "" {
		chClient, err = clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.URL,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}

		sentinelRequests = clickhouse.NewBuffer[schema.SentinelRequest](chClient, "default.sentinel_requests_raw_v1", clickhouse.BufferConfig{
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

		// Close buffers before connection (LIFO)
		r.Defer(func() error { sentinelRequests.Close(); return nil })
		r.Defer(func() error { keyVerifications.Close(); return nil })
		r.Defer(chClient.Close)
	}

	// Initialize gossip-based cache invalidation
	var broadcaster clustering.Broadcaster
	if cfg.Gossip != nil {
		logger.Info("Initializing gossip cluster for cache invalidation",
			"region", cfg.Region,
			"instanceID", cfg.SentinelID,
		)

		mux := cluster.NewMessageMux()

		lanSeeds := cluster.ResolveDNSSeeds(cfg.Gossip.LANSeeds, cfg.Gossip.LANPort)
		wanSeeds := cluster.ResolveDNSSeeds(cfg.Gossip.WANSeeds, cfg.Gossip.WANPort)

		gossipCluster, clusterErr := cluster.New(cluster.Config{
			Region:           cfg.Region,
			NodeID:           cfg.SentinelID,
			BindAddr:         cfg.Gossip.BindAddr,
			BindPort:         cfg.Gossip.LANPort,
			WANBindPort:      cfg.Gossip.WANPort,
			WANAdvertiseAddr: cfg.Gossip.WANAdvertiseAddr,
			LANSeeds:         lanSeeds,
			WANSeeds:         wanSeeds,
			SecretKey:        nil, // Sentinel gossip is locked down via CiliumNetworkPolicy
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

	routerSvc, err := router.New(router.Config{
		DB:            database,
		Clock:         clk,
		EnvironmentID: cfg.EnvironmentID,
		Platform:      cfg.Platform,
		Region:        cfg.Region,
		Broadcaster:   broadcaster,
		NodeID:        cfg.SentinelID,
	})
	if err != nil {
		return fmt.Errorf("unable to create router service: %w", err)
	}
	r.Defer(routerSvc.Close)

	middlewareEngine, err := initMiddlewareEngine(middlewareEngineCfg{
		Runner:           r,
		RedisURL:         cfg.Redis.URL,
		Region:           cfg.Region,
		Database:         database,
		KeyVerifications: keyVerifications,
		Clock:            clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create middleware engine: %w", err)
	}

	// Workspace quota cache for retention stamping. Sentinel doesn't run
	// the rate-limit-from-quota path so we wire a small dedicated cache
	// instead of hijacking the keys service's QuotaCache slot.
	quotaCache, err := cache.New(cache.Config[string, keysdb.Quotas]{
		Fresh:    1 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10_000,
		Resource: "sentinel_workspace_quota",
		Clock:    clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create quota cache: %w", err)
	}

	svcs := &routes.Services{
		RouterService:      routerSvc,
		Clock:              clk,
		WorkspaceID:        cfg.WorkspaceID,
		EnvironmentID:      cfg.EnvironmentID,
		SentinelID:         cfg.SentinelID,
		Region:             cfg.Region,
		Platform:           cfg.Platform,
		SentinelRequests:   sentinelRequests,
		MaxRequestBodySize: maxRequestBodySize,
		RequestTimeout:     cfg.RequestTimeout,
		Engine:             middlewareEngine,
		Pprof:              cfg.Pprof,
	}

	srv, err := zen.New(zen.Config{
		TLS:                nil,
		Flags:              nil,
		EnableH2C:          true,
		MaxRequestBodySize: maxRequestBodySize,
		ReadTimeout:        -1,
		WriteTimeout:       -1,
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)
	}
	// Sentinel-owned quota cache + resolver for per-row expires_at stamping.
	srv.SetLogsRetentionResolver(quotaretention.New(quotaCache, database))
	r.RegisterHealth(srv.Mux(), "/_unkey/internal/health")
	r.AddReadinessCheck("database", func(ctx context.Context) error {
		return database.RW().PingContext(ctx)
	})
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

// initMiddlewareEngine creates the middleware engine for policy evaluation.
// Redis is not a critical dependency: if a Redis URL is configured it is used
// for distributed rate limiting and usage tracking; otherwise an in-memory
// counter is used as fallback. Even with Redis configured, a connection failure
// at startup does not prevent the engine from being created, the Redis client
// reconnects lazily, and both the rate limiter and usage limiter degrade
// gracefully (local windows / DB fallback) when Redis is temporarily unavailable.
func initMiddlewareEngine(cfg middlewareEngineCfg) (engine.Evaluator, error) {
	var ctr counter.Counter
	if cfg.RedisURL != "" {
		redisCtr, err := counter.NewRedis(counter.RedisConfig{
			RedisURL: cfg.RedisURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create redis counter: %w", err)
		}
		cfg.Runner.Defer(redisCtr.Close)
		ctr = redisCtr
	} else {
		memoryCtr := counter.NewMemory()
		cfg.Runner.Defer(memoryCtr.Close)
		ctr = memoryCtr
	}

	rlSvc, err := ratelimit.New(ratelimit.Config{
		Clock:   cfg.Clock,
		Counter: ctr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ratelimit service: %w", err)
	}
	cfg.Runner.Defer(rlSvc.Close)

	usageLimiter, err := usagelimiter.NewCounter(usagelimiter.CounterConfig{
		FindKeyCredits: func(ctx context.Context, keyID string) (int64, bool, error) {
			limit, err := db.WithRetryContext(ctx, func() (sql.NullInt64, error) {
				return db.Query.FindKeyCredits(ctx, cfg.Database.RO(), keyID)
			})
			if err != nil {
				return 0, false, err
			}
			return limit.Int64, limit.Valid, nil
		},
		DecrementKeyCredits: func(ctx context.Context, keyID string, cost int64) error {
			return db.Query.UpdateKeyCreditsDecrement(ctx, cfg.Database.RW(), db.UpdateKeyCreditsDecrementParams{
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
	cfg.Runner.Defer(usageLimiter.Close)

	keyCache, err := cache.New(cache.Config[string, keysdb.CachedKeyData]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		MaxSize:  100_000,
		Resource: "sentinel_key_cache",
		Clock:    cfg.Clock,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create key cache: %w", err)
	}

	keyService, err := keys.New(keys.Config{
		DB:               db.ToMySQL(cfg.Database),
		RateLimiter:      rlSvc,
		RBAC:             rbac.New(),
		KeyVerifications: cfg.KeyVerifications,
		Region:           cfg.Region,
		UsageLimiter:     usageLimiter,
		KeyCache:         keyCache,
		QuotaCache:       nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create key service: %w", err)
	}

	logger.Info("middleware engine initialized")
	eng, err := engine.New(engine.Config{
		KeyService:  keyService,
		RateLimiter: rlSvc,
		Clock:       cfg.Clock,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize middleware engine: %w", err)
	}
	return eng, nil
}
