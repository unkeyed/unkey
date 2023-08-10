package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	cacheMiddleware "github.com/unkeyed/unkey/apps/agent/pkg/cache/middleware"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/env"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tinybird"

	"os"
	"os/signal"
	"syscall"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/kafka"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/version"
	"go.uber.org/zap"
)

var (
	enableAxiom    bool
	enableTinybird bool
)

func init() {
	AgentCmd.Flags().BoolVar(&enableAxiom, "enable-axiom", false, "Send logs and traces to axiom")
	AgentCmd.Flags().BoolVar(&enableTinybird, "enable-tinybird", false, "Send analytics to tinybird")
}

// AgentCmd represents the agent command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run the Unkey agent",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		logger := logging.New().With(zap.String("version", version.Version), zap.String("allocId", os.Getenv("FLY_ALLOC_ID")))

		defer func() {
			// this is best effort and can error quite frequently
			_ = logger.Sync()
		}()
		logger.Info("Starting Unkey Agent")

		e := env.Env{
			ErrorHandler: func(err error) { logger.Fatal("unable to load environment variable", zap.Error(err)) },
		}
		region := e.String("FLY_REGION", "local")
		logger = logger.With(zap.String("region", region))

		var tb *tinybird.Tinybird
		if enableTinybird {
			tinybirdToken := e.String("TINYBIRD_TOKEN")
			if tinybirdToken != "" {
				tb = tinybird.New(tinybird.Config{
					Token:  tinybirdToken,
					Logger: logger,
				})
				defer tb.Close()
			}
		}

		var tracer tracing.Tracer
		if enableAxiom {
			t, closeTracer, err := tracing.New(context.Background(), tracing.Config{
				Dataset:    "tracing",
				Service:    "api",
				Version:    version.Version,
				AxiomOrgId: e.String("AXIOM_ORG_ID"),
				AxiomToken: e.String("AXIOM_TOKEN"),
			})
			if err != nil {
				logger.Fatal("unable to start tracer", zap.Error(err))
			}
			defer func() {
				err := closeTracer()
				if err != nil {
					logger.Fatal("unable to close tracer", zap.Error(err))
				}
			}()
			tracer = t
			logger.Info("Axiom tracing enabled")
		} else {
			tracer = tracing.NewNoop()
		}

		k, err := kafka.New(kafka.Config{
			Logger:   logger,
			GroupId:  e.String("FLY_ALLOC_ID", "local"),
			Broker:   e.String("KAFKA_BROKER"),
			Username: e.String("KAFKA_USERNAME"),
			Password: e.String("KAFKA_PASSWORD"),
		})
		if err != nil {
			logger.Fatal("unable to start kafka", zap.Error(err))
		}

		go k.Start()
		defer k.Close()

		fastRatelimit := ratelimit.NewInMemory()
		var consistentRatelimit ratelimit.Ratelimiter
		redisUrl := e.String("REDIS_URL", "")
		if redisUrl != "" {
			consistentRatelimit, err = ratelimit.NewRedis(ratelimit.RedisConfig{
				RedisUrl: redisUrl,
			})
			if err != nil {
				logger.Fatal("unable to start redis ratelimiting", zap.Error(err))
			}
		}

		db, err := database.New(database.Config{
			Logger:           logger,
			PrimaryUs:        e.String("DATABASE_DSN"),
			ReplicaEu:        e.String("DATABASE_DSN_EU", ""),
			ReplicaAsia:      e.String("DATABASE_DSN_ASIA", ""),
			FlyRegion:        region,
			PlanetscaleBoost: e.Bool("PLANETSCALE_BOOST", false),
		},
			database.WithTracing(tracer))
		if err != nil {
			logger.Fatal("Failed to connect to database", zap.Error(err))
		}

		keyCache := cache.New[entities.Key](cache.Config[entities.Key]{
			Fresh:             time.Minute,
			Stale:             time.Minute * 15,
			RefreshFromOrigin: db.FindKeyByHash,
			Logger:            logger,
		})
		keyCache = cacheMiddleware.WithTracing[entities.Key](keyCache, tracer)
		keyCache = cacheMiddleware.WithLogging[entities.Key](keyCache, logger.With(zap.String("cacheType", "key")))

		apiCache := cache.New[entities.Api](cache.Config[entities.Api]{
			Fresh:             time.Minute,
			Stale:             time.Minute * 15,
			RefreshFromOrigin: db.FindApi,
			Logger:            logger,
		})
		apiCache = cacheMiddleware.WithTracing[entities.Api](apiCache, tracer)
		apiCache = cacheMiddleware.WithLogging[entities.Api](apiCache, logger.With(zap.String("cacheType", "api")))

		k.RegisterOnKeyEvent(func(ctx context.Context, e kafka.KeyEvent) error {
			logger.Info("evicting key from cache", zap.String("keyId", e.Key.Id), zap.String("keyHash", e.Key.Hash))
			keyCache.Remove(context.Background(), e.Key.Hash)

			if e.Type == kafka.KeyCreated || e.Type == kafka.KeyUpdated {
				logger.Info("fetching key from origin", zap.String("keyId", e.Key.Id), zap.String("keyHash", e.Key.Hash))
				key, found, err := db.FindKeyById(ctx, e.Key.Id)
				if err != nil {
					return fmt.Errorf("unable to get key by id: %s: %w", e.Key.Id, err)
				}
				if found {
					keyCache.Set(ctx, key.Hash, key)
				}
			}

			return nil
		})

		port := e.String("PORT", "8080")

		srv := server.New(server.Config{
			Logger:            logger,
			KeyCache:          keyCache,
			ApiCache:          apiCache,
			Database:          db,
			Ratelimit:         fastRatelimit,
			GlobalRatelimit:   consistentRatelimit,
			Tracer:            tracer,
			Tinybird:          tb,
			UnkeyAppAuthToken: e.String("UNKEY_APP_AUTH_TOKEN"),
			UnkeyWorkspaceId:  e.String("UNKEY_WORKSPACE_ID"),
			UnkeyApiId:        e.String("UNKEY_API_ID"),
			UnkeyKeyAuthId:    e.String("UNKEY_KEY_AUTH_ID"),
			Region:            region,
			Kafka:             k,
			Version:           version.Version,
		})

		go func() {
			err = srv.Start(fmt.Sprintf("0.0.0.0:%s", port))
			if err != nil {
				logger.Fatal("Failed to run service", zap.Error(err))
			}
		}()
		defer srv.Close()

		cShutdown := make(chan os.Signal, 1)
		signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

		// wait for signal
		sig := <-cShutdown
		logger.Info("Caught signal, shutting down", zap.Any("sig", sig))
	},
}
