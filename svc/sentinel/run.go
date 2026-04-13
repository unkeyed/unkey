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
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/batch"
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
	"github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
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
			Application:     "sentinel",
			Version:         version.Version,
			InstanceID:      cfg.SentinelID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.Observability.Tracing.SampleRate,
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
		slog.String("version", version.Version),
	))

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	lazy.SetRegistry(reg)

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

	ctr, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: cfg.Redis.URL,
	})
	if err != nil {
		return fmt.Errorf("unable to create counter: %w", err)
	}
	r.Defer(ctr.Close)

	rlSvc, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: ctr,
	})
	if err != nil {
		return fmt.Errorf("unable to create ratelimit service: %w", err)
	}
	r.Defer(rlSvc.Close)

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

	// Initialize middleware engine for KeyAuth and other sentinel policies.
	// Uses Redis if configured, in-memory counters otherwise.
	middlewareEngine, err := initMiddlewareEngine(r, cfg.Redis.URL, cfg.Region, rlSvc, database, keyVerifications, clk)
	if err != nil {
		return fmt.Errorf("unable to create middleware engine: %w", err)
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
// Redis is not a critical dependency: if a Redis URL is configured, it is used
// for distributed rate limiting and usage tracking; otherwise an in-memory
// counter is used as fallback. Even with Redis configured, a connection failure
// at startup does not prevent the engine from being created — the Redis client
// reconnects lazily, and both the rate limiter and usage limiter degrade
// gracefully (local windows / DB fallback) when Redis is temporarily unavailable.
func initMiddlewareEngine(r *runner.Runner, redisURL string, region string, rlSvc ratelimit.Service, database db.Database, keyVerifications *batch.BatchProcessor[schema.KeyVerification], clk clock.Clock) (engine.Evaluator, error) {
	var ctr counter.Counter
	if redisURL != "" {
		redisCtr, redisErr := counter.NewRedis(counter.RedisConfig{
			RedisURL: redisURL,
		})
		if redisErr != nil {
			return nil, fmt.Errorf("failed to create redis counter: %w", redisErr)
		}
		r.Defer(redisCtr.Close)
		ctr = redisCtr
		logger.Info("middleware engine using redis counter")
	} else {
		ctr = counter.NewMemory()
		r.Defer(ctr.Close)
		logger.Info("redis URL not configured, middleware engine using in-memory counter")
	}

	rateLimiter, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: ctr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limiter: %w", err)
	}
	r.Defer(rateLimiter.Close)

	usageLimiter, err := usagelimiter.NewCounter(usagelimiter.CounterConfig{
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
		Resource: "sentinel_key_cache",
		Clock:    clk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create key cache: %w", err)
	}

	keyService, err := keys.New(keys.Config{
		DB:               db.ToMySQL(database),
		RateLimiter:      rateLimiter,
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

	logger.Info("middleware engine initialized")
	return engine.New(engine.Config{
		KeyService:  keyService,
		RateLimiter: rlSvc,
		Clock:       clk,
	}), nil
}
