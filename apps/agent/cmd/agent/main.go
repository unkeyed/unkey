package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	analyticsMiddleware "github.com/unkeyed/unkey/apps/agent/pkg/analytics/middleware"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics/tinybird"
	"github.com/unkeyed/unkey/apps/agent/pkg/boot"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	cacheMiddleware "github.com/unkeyed/unkey/apps/agent/pkg/cache/middleware"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/env"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	metricsPkg "github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"

	"os"
	"os/signal"
	"syscall"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/events/kafka"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/version"
)

type features struct {
	enableAxiom bool
	analytics   string
	eventBus    string
	prewarm     bool
	verbose     bool
}

var runtimeConfig features

func init() {
	AgentCmd.Flags().BoolVar(&runtimeConfig.enableAxiom, "enable-axiom", false, "Send logs and traces to axiom")
	AgentCmd.Flags().StringVar(&runtimeConfig.analytics, "analytics", "", "Send analytics to a backend, available: ['tinybird']")
	AgentCmd.Flags().StringVar(&runtimeConfig.eventBus, "event-bus", "", "Use a message bus for communication between nodes, available: ['kafka']")
	AgentCmd.Flags().BoolVar(&runtimeConfig.prewarm, "prewarm", false, "Load all keys from the db to memory on boot")
	AgentCmd.Flags().BoolVarP(&runtimeConfig.verbose, "verbose", "v", false, "Print debug logs")
}

// AgentCmd represents the agent command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run the Unkey agent",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		e := env.Env{
			ErrorHandler: func(err error) {
				log.Fatalf("unable to load environment variable: %s", err.Error())
			},
		}
		logConfig := &logging.Config{
			Debug: runtimeConfig.verbose,
		}
		if runtimeConfig.enableAxiom {
			axiomWriter, err := logging.NewAxiomWriter(logging.AxiomWriterConfig{
				AxiomToken: e.String("AXIOM_TOKEN"),
				AxiomOrgId: e.String("AXIOM_ORG_ID"),
			})
			if err != nil {
				log.Fatalf("unable to create axiom writer: %s", err.Error())
			}
			logConfig.Writer = append(logConfig.Writer, axiomWriter)
		}

		logger, err := logging.New(logConfig)
		if err != nil {
			log.Fatalf("unable to create logger: %s", err.Error())
		}
		logger = logger.With().Str("version", version.Version).Logger()

		logger.Info().Any("runtimeConfig", runtimeConfig).Msg("Starting Unkey Agent")

		region := e.String("FLY_REGION", "local")
		allocId := e.String("FLY_ALLOC_ID", "")
		logger = logger.With().Str("region", region).Logger()
		if allocId != "" {
			logger = logger.With().Str("allocId", allocId).Logger()
		}

		metrics := metricsPkg.NewNoop()
		if runtimeConfig.enableAxiom {
			realMetrics, err := metricsPkg.New(metricsPkg.Config{
				AxiomOrgId: e.String("AXIOM_ORG_ID"),
				AxiomToken: e.String("AXIOM_TOKEN"),
				Logger:     logger.With().Str("pkg", "metrics").Logger(),
				Region:     region,
			})
			if err != nil {
				logger.Fatal().Err(err).Msg("unable to start metrics")
			}
			metrics = realMetrics
		}
		defer metrics.Close()

		// Setup Axiom

		tracer := tracing.NewNoop()
		{
			if runtimeConfig.enableAxiom {
				t, closeTracer, err := tracing.New(context.Background(), tracing.Config{
					Dataset:    "tracing",
					Service:    "agent",
					Version:    version.Version,
					AxiomOrgId: e.String("AXIOM_ORG_ID"),
					AxiomToken: e.String("AXIOM_TOKEN"),
				})
				if err != nil {
					logger.Fatal().Err(err).Msg("unable to start tracer")
				}
				defer func() {
					err := closeTracer()
					if err != nil {
						logger.Fatal().Err(err).Msg("unable to close tracer")
					}
				}()
				tracer = t
				logger.Info().Msg("Axiom tracing enabled")
			}
		}

		db, err := database.New(
			database.Config{
				Logger:      logger,
				PrimaryUs:   e.String("DATABASE_DSN"),
				ReplicaEu:   e.String("DATABASE_DSN_EU", ""),
				ReplicaAsia: e.String("DATABASE_DSN_ASIA", ""),
				FlyRegion:   region,
			},
			database.WithMetrics(metrics),
			database.WithTracing(tracer),
		)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to connect to database")
		}
		defer db.Close()
		// Setup Analytics

		a := analytics.NewNoop()
		{
			if runtimeConfig.analytics == "tinybird" {
				tb := tinybird.New(tinybird.Config{
					Token:  e.String("TINYBIRD_TOKEN"),
					Logger: logger,
				})
				defer tb.Close()
				a = tb
			}
			a = analyticsMiddleware.WithTracing(a, tracer)

		}

		// Setup Event Bus

		eventBus := events.NewNoop()
		{
			if runtimeConfig.eventBus == "kafka" {
				k, err := kafka.New(kafka.Config{
					Logger:   logger,
					GroupId:  e.String("FLY_ALLOC_ID", "local"),
					Broker:   e.String("KAFKA_BROKER"),
					Username: e.String("KAFKA_USERNAME"),
					Password: e.String("KAFKA_PASSWORD"),
				})
				if err != nil {
					logger.Fatal().Err(err).Msg("unable to start kafka")
				}

				k.Start()
				defer k.Close()
				eventBus = k
			}
		}

		fastRatelimit := ratelimit.NewInMemory()
		var consistentRatelimit ratelimit.Ratelimiter
		redisUrl := e.String("REDIS_URL", "")
		if redisUrl != "" {
			consistentRatelimit, err = ratelimit.NewRedis(ratelimit.RedisConfig{
				RedisUrl: redisUrl,
			})
			if err != nil {
				logger.Fatal().Err(err).Msg("unable to start redis ratelimiting")
			}
		}

		keyCache := cache.New[entities.Key](cache.Config[entities.Key]{
			Fresh:   time.Minute * 15,
			Stale:   time.Minute * 60,
			MaxSize: 1024 * 1024,
			RefreshFromOrigin: func(ctx context.Context, keyHash string) (entities.Key, bool) {
				key, found, err := db.FindKeyByHash(ctx, keyHash)
				if err != nil {
					logger.Err(err).Msg("unable to refresh key by hash")
					return entities.Key{}, false
				}
				return key, found
			},
			Logger:   logger.With().Str("cacheType", "key").Logger(),
			Metrics:  metrics,
			Resource: "key",
		})
		keyCache = cacheMiddleware.WithTracing[entities.Key](keyCache, tracer)
		keyCache = cacheMiddleware.WithMetrics[entities.Key](keyCache, metrics, "key")

		apiByKeyAuthIdCache := cache.New[entities.Api](cache.Config[entities.Api]{
			Fresh:   time.Minute * 5,
			Stale:   time.Minute * 15,
			MaxSize: 1024 * 1024,
			RefreshFromOrigin: func(ctx context.Context, keyAuthId string) (entities.Api, bool) {
				api, found, err := db.FindApiByKeyAuthId(ctx, keyAuthId)
				if err != nil {
					logger.Err(err).Msg("unable to refresh api by keyAuthId")
					return entities.Api{}, false
				}
				return api, found
			},
			Logger:   logger.With().Str("cacheType", "api").Logger(),
			Metrics:  metrics,
			Resource: "api",
		})
		apiByKeyAuthIdCache = cacheMiddleware.WithTracing[entities.Api](apiByKeyAuthIdCache, tracer)
		apiByKeyAuthIdCache = cacheMiddleware.WithMetrics[entities.Api](apiByKeyAuthIdCache, metrics, "api")

		eventBus.OnKeyEvent(func(ctx context.Context, e events.KeyEvent) error {

			if e.Type == events.KeyDeleted {
				logger.Debug().Str("keyId", e.Key.Id).Str("keyHash", e.Key.Hash).Msg("evicting from cache")
				keyCache.Remove(context.Background(), e.Key.Hash)
				return nil
			}

			logger.Debug().Str("keyId", e.Key.Id).Str("keyHash", e.Key.Hash).Msg("precaching key")
			key, found, err := cache.WithCache(keyCache, db.FindKeyById)(ctx, e.Key.Id)
			if err != nil {
				return fmt.Errorf("unable to get key by id: %s: %w", e.Key.Id, err)
			}
			if !found {
				return nil
			}
			logger.Debug().Str("keyAuthId", key.KeyAuthId).Msg("precaching api")

			_, _, err = cache.WithCache(apiByKeyAuthIdCache, db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
			if err != nil {
				return fmt.Errorf("unable to find api by keyAuthId: %s: %w", key.KeyAuthId, err)
			}

			return nil
		})

		if runtimeConfig.prewarm {

			cacheWarmer := boot.NewCacheWarmer(boot.Config{
				KeyCache: keyCache,
				ApiCache: apiByKeyAuthIdCache,
				DB:       db,
				Logger:   logger,
			})
			defer cacheWarmer.Stop()

			go func() {
				err := cacheWarmer.Run(context.Background())
				if err != nil {
					logger.Err(err).Msg("unable to warm cache")
				}
			}()
		}

		workspaceService := workspaces.New(
			workspaces.Config{
				Database: db,
			},
			workspaces.WithLogging(logger),
			workspaces.WithTracing(tracer),
		)

		apiService := apis.New(apis.Config{
			Database: db,
		},
			apis.WithLogging(logger),
			apis.WithTracing(tracer),
		)

		srv := server.New(server.Config{
			Logger:            logger,
			KeyCache:          keyCache,
			ApiCache:          apiByKeyAuthIdCache,
			Database:          db,
			Ratelimit:         fastRatelimit,
			GlobalRatelimit:   consistentRatelimit,
			Tracer:            tracer,
			Analytics:         a,
			UnkeyAppAuthToken: e.String("UNKEY_APP_AUTH_TOKEN"),
			UnkeyWorkspaceId:  e.String("UNKEY_WORKSPACE_ID"),
			UnkeyApiId:        e.String("UNKEY_API_ID"),
			UnkeyKeyAuthId:    e.String("UNKEY_KEY_AUTH_ID"),
			Region:            region,
			EventBus:          eventBus,
			Version:           version.Version,
			WorkspaceService:  workspaceService,
			ApiService:        apiService,
			Metrics:           metrics,
		})

		go func() {
			err = srv.Start(fmt.Sprintf("0.0.0.0:%s", e.String("PORT", "8080")))
			if err != nil {
				logger.Fatal().Err(err).Msg("Failed to run service")
			}
		}()
		defer srv.Close()

		cShutdown := make(chan os.Signal, 1)
		signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

		// wait for signal
		sig := <-cShutdown
		logger.Info().Any("sig", sig).Msg("Caught signal, shutting down")
	},
}
