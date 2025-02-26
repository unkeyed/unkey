package api

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/cmd/api/routes"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/aws/ecs"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/config"
	"github.com/unkeyed/unkey/go/pkg/database"
	dbCache "github.com/unkeyed/unkey/go/pkg/database/middleware/cache"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:        "api",
	Description: "Run the API server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Usage:       "Load configuration file",
			Value:       "unkey.json",
			DefaultText: "unkey.json",
			EnvVars:     []string{"UNKEY_CONFIG_FILE"},
		},
		&cli.BoolFlag{
			Name: "generate-config-schema",
		},
	},
	Action: run,
}

// nolint:gocognit
func run(cliC *cli.Context) error {

	shutdowns := []shutdown.ShutdownFn{}

	if cliC.Bool("generate-config-schema") {
		// nolint:exhaustruct
		_, err := config.GenerateJsonSchema(nodeConfig{}, "schema.json")
		if err != nil {
			panic(err)
		}

		fmt.Println("Schema generated successfully and written to schema.json")

		return nil
	}
	ctx := cliC.Context
	clk := clock.New()
	configFile := cliC.String("config")

	// nolint:exhaustruct
	cfg := nodeConfig{}
	err := config.LoadFile(&cfg, configFile)
	if err != nil {
		return fmt.Errorf("unable to load config file: %w", err)
	}

	nodeID := uid.Node()
	if cfg.Cluster != nil {
		if cfg.Cluster.NodeID == "" {
			cfg.Cluster.NodeID = nodeID
		} else {
			nodeID = cfg.Cluster.NodeID
		}
	}

	if cfg.Region == "" {
		cfg.Region = "unknown"
	}
	logger := logging.New(logging.Config{Development: true, NoColor: true}).
		With(
			slog.String("nodeId", nodeID),
			slog.String("platform", cfg.Platform),
			slog.String("region", cfg.Region),
			slog.String("version", version.Version),
		)

	logger.Info(ctx, "env", slog.String("env", strings.Join(os.Environ(), "\n")))
	// Catch any panics now after we have a logger but before we start the server
	defer func() {
		if r := recover(); r != nil {
			logger.Error(ctx, "panic",
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()

	logger.Info(ctx, "configration loaded", slog.String("file", configFile))

	if cfg.Otel != nil {
		shutdownOtel, grafanaErr := otel.InitGrafana(ctx, otel.Config{
			GrafanaEndpoint: cfg.Otel.OtlpEndpoint,
			Application:     "api",
			Version:         version.Version,
		})
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
		shutdowns = append(shutdowns, shutdownOtel...)
	}

	db, err := database.New(database.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
		Logger:      logger,
		Clock:       clock.New(),
	}, dbCache.WithCaching(logger))
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	defer db.Close()

	c, shutdownCluster, err := setupCluster(cfg, logger)
	if err != nil {
		return fmt.Errorf("unable to create cluster: %w", err)
	}
	shutdowns = append(shutdowns, shutdownCluster...)

	var ch clickhouse.Bufferer = clickhouse.NewNoop()
	if cfg.Clickhouse != nil {
		ch, err = clickhouse.New(clickhouse.Config{
			URL:    cfg.Clickhouse.Url,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
	}

	srv, err := zen.New(zen.Config{
		NodeID: nodeID,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)
	}

	validator, err := validation.New()
	if err != nil {
		return fmt.Errorf("unable to create validator: %w", err)
	}

	keySvc, err := keys.New(keys.Config{
		Logger: logger,
		DB:     db,
	})
	if err != nil {
		return fmt.Errorf("unable to create key service: %w", err)
	}

	rlSvc, err := ratelimit.New(ratelimit.Config{
		Logger:  logger,
		Cluster: c,
		Clock:   clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create ratelimit service: %w", err)
	}

	routes.Register(srv, &routes.Services{
		Logger:      logger,
		Database:    db,
		EventBuffer: ch,
		Keys:        keySvc,
		Validator:   validator,
		Ratelimit:   rlSvc,
	})

	go func() {
		listenErr := srv.Listen(ctx, fmt.Sprintf(":%d", cfg.HttpPort))
		if listenErr != nil {
			panic(listenErr)
		}
	}()

	return gracefulShutdown(ctx, logger, shutdowns)
}

func gracefulShutdown(ctx context.Context, logger logging.Logger, shutdowns []shutdown.ShutdownFn) error {
	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	<-cShutdown
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	logger.Info(ctx, "shutting down")

	errors := []error{}
	for i := len(shutdowns) - 1; i >= 0; i-- {
		err := shutdowns[i](ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred during shutdown: %v", errors)
	}
	return nil
}

func setupCluster(cfg nodeConfig, logger logging.Logger) (cluster.Cluster, []shutdown.ShutdownFn, error) {
	shutdowns := []shutdown.ShutdownFn{}
	if cfg.Cluster == nil {
		return cluster.NewNoop("", "127.0.0.1"), shutdowns, nil
	}

	var advertiseAddr string
	{
		switch {
		case cfg.Cluster.AdvertiseAddr.Static != nil:
			{
				advertiseAddr = *cfg.Cluster.AdvertiseAddr.Static
			}
		case cfg.Cluster.AdvertiseAddr.AwsEcsMetadata != nil && *cfg.Cluster.AdvertiseAddr.AwsEcsMetadata:

			{
				var getDnsErr error
				advertiseAddr, getDnsErr = ecs.GetPrivateDnsName()
				if getDnsErr != nil {
					return nil, shutdowns, fmt.Errorf("unable to get private dns name: %w", getDnsErr)
				}

			}
		default:
			return nil, shutdowns, fmt.Errorf("invalid advertise address configuration")
		}
	}

	gossipPort, err := strconv.ParseInt(cfg.Cluster.GossipPort, 10, 64)
	if err != nil {
		return nil, shutdowns, fmt.Errorf("unable to parse gossip port: %w", err)
	}

	m, err := membership.New(membership.Config{
		NodeID:     cfg.Cluster.NodeID,
		Addr:       advertiseAddr,
		GossipPort: int(gossipPort),
		Logger:     logger,
	})
	if err != nil {
		return nil, shutdowns, fmt.Errorf("unable to create membership: %w", err)
	}

	rpcPort, err := strconv.ParseInt(cfg.Cluster.RpcPort, 10, 64)
	if err != nil {
		return nil, shutdowns, fmt.Errorf("unable to parse rpc port: %w", err)
	}
	c, err := cluster.New(cluster.Config{
		Self: cluster.Node{

			ID:      cfg.Cluster.NodeID,
			Addr:    advertiseAddr,
			RpcAddr: "TO DO",
		},
		Logger:     logger,
		Membership: m,
		RpcPort:    int(rpcPort),
	})
	if err != nil {
		return nil, shutdowns, fmt.Errorf("unable to create cluster: %w", err)
	}
	shutdowns = append(shutdowns, c.Shutdown)

	var d discovery.Discoverer

	switch {
	case cfg.Cluster.Discovery.Static != nil:
		{
			d = &discovery.Static{
				Addrs: cfg.Cluster.Discovery.Static.Addrs,
			}
			break
		}

	case cfg.Cluster.Discovery.Redis != nil:
		{
			rd, rErr := discovery.NewRedis(discovery.RedisConfig{
				URL:    cfg.Cluster.Discovery.Redis.URL,
				NodeID: cfg.Cluster.NodeID,
				Addr:   advertiseAddr,
				Logger: logger,
			})
			if rErr != nil {
				return nil, shutdowns, fmt.Errorf("unable to create redis discovery: %w", rErr)
			}
			shutdowns = append(shutdowns, rd.Shutdown)
			d = rd
			break
		}
	default:
		{
			return nil, nil, fmt.Errorf("missing discovery method")
		}
	}

	err = m.Start(d)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to start membership: %w", err)
	}

	return c, shutdowns, nil
}
