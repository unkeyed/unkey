package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/cmd/api/routes"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/config"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/membership"
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
	configFile := cliC.String("config")

	// nolint:exhaustruct
	cfg := nodeConfig{}
	err := config.LoadFile(&cfg, configFile)
	if err != nil {
		return fmt.Errorf("unable to load config file: %w", err)
	}

	nodeID := uid.Node()
	if cfg.Cluster != nil && cfg.Cluster.NodeID != "" {
		nodeID = cfg.Cluster.NodeID
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

	var c cluster.Cluster = cluster.NewNoop(nodeID, net.ParseIP("127.0.0.1"))
	if cfg.Cluster != nil {
		var d discovery.Discoverer

		switch {
		case cfg.Cluster.Discovery.Static != nil:

			d = &discovery.Static{
				Addrs: cfg.Cluster.Discovery.Static.Addrs,
			}
		case cfg.Cluster.Discovery.AwsCloudmap != nil:
			return fmt.Errorf("NOT IMPLEMENTED")
		default:
			return fmt.Errorf("missing discovery method")
		}

		gossipPort, err := strconv.ParseInt(cfg.Cluster.GossipPort, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse gossip port: %w", err)
		}

		m, mErr := membership.New(membership.Config{
			NodeID:     nodeID,
			Addr:       net.ParseIP(""),
			GossipPort: int(gossipPort),
			Logger:     logger,
		})
		if mErr != nil {
			return fmt.Errorf("unable to create membership: %w", err)
		}

		rpcPort, err := strconv.ParseInt(cfg.Cluster.RpcPort, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse rpc port: %w", err)
		}
		c, err = cluster.New(cluster.Config{
			Self: cluster.Node{

				ID:      nodeID,
				Addr:    net.ParseIP(cfg.Cluster.AdvertiseAddr),
				RpcAddr: "TO DO",
			},
			Logger:     logger,
			Membership: m,
			RpcPort:    int(rpcPort),
		})
		if err != nil {
			return fmt.Errorf("unable to create cluster: %w", err)
		}

		err = m.Start(d)
		if err != nil {
			return fmt.Errorf("unable to start membership: %w", err)
		}
	}

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

	db, err := database.New(database.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
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

	routes.Register(srv, &routes.Services{
		Logger:      logger,
		Database:    db,
		EventBuffer: ch,
		Keys:        keySvc,
		Validator:   validator,
	})

	go func() {
		listenErr := srv.Listen(ctx, fmt.Sprintf(":%d", cfg.HttpPort))
		if listenErr != nil {
			panic(listenErr)
		}
	}()

	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	<-cShutdown
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	logger.Info(ctx, "shutting down")
	err = c.Shutdown(ctx)

	if err != nil {
		return fmt.Errorf("unable to leave cluster: %w", err)
	}
	return nil
}
