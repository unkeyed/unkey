package sentinel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/cache"
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

	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {
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

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}
	r.Defer(database.Close)

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickHouse.URL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.URL,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
		r.Defer(ch.Close)
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
	// When Redis URL is empty: nil engine, pass-through (no policies expected).
	var middlewareEngine engine.Evaluator
	if cfg.Redis.URL == "" {
		logger.Info("redis URL not configured, middleware engine disabled")
	} else {
		eng, closers, initErr := initMiddlewareEngine(cfg, database, ch, clk)
		if initErr != nil {
			return fmt.Errorf("unable to create middleware engine: %w", initErr)
		}
		r.Defer(closers...)
		middlewareEngine = eng
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

// initMiddlewareEngine creates the middleware engine backed by Redis.
// Returns the engine and a slice of closer functions to be deferred on success.
// On failure, any resources created so far are closed before returning the error.
func initMiddlewareEngine(cfg Config, database db.Database, ch clickhouse.ClickHouse, clk clock.Clock) (engine.Evaluator, []runner.CloseFunc, error) {
	var closers []runner.CloseFunc

	closeAll := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}

	redisCounter, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: cfg.Redis.URL,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	closers = append(closers, redisCounter.Close)

	rateLimiter, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: redisCounter,
	})
	if err != nil {
		closeAll()
		return nil, nil, fmt.Errorf("failed to create rate limiter: %w", err)
	}
	closers = append(closers, rateLimiter.Close)

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
		Counter:       redisCounter,
		TTL:           60 * time.Second,
		ReplayWorkers: 8,
	})
	if err != nil {
		closeAll()
		return nil, nil, fmt.Errorf("failed to create usage limiter: %w", err)
	}
	closers = append(closers, usageLimiter.Close)

	keyCache, err := cache.New[string, db.CachedKeyData](cache.Config[string, db.CachedKeyData]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		MaxSize:  100_000,
		Resource: "sentinel_key_cache",
		Clock:    clk,
	})
	if err != nil {
		closeAll()
		return nil, nil, fmt.Errorf("failed to create key cache: %w", err)
	}

	keyService, err := keys.New(keys.Config{
		DB:           database,
		RateLimiter:  rateLimiter,
		RBAC:         rbac.New(),
		Clickhouse:   ch,
		Region:       cfg.Region,
		UsageLimiter: usageLimiter,
		KeyCache:     keyCache,
		QuotaCache:   nil,
	})
	if err != nil {
		closeAll()
		return nil, nil, fmt.Errorf("failed to create key service: %w", err)
	}

	logger.Info("middleware engine initialized")
	eng := engine.New(engine.Config{
		KeyService: keyService,
		Clock:      clk,
	})
	return eng, closers, nil
}
