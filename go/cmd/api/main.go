package api

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/config"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name: "api",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Usage:       "Load configuration file",
			Value:       "unkey.json",
			DefaultText: "unkey.json",
			EnvVars:     []string{"AGENT_CONFIG_FILE"},
		},
	},
	Action: run,
}

func run(cliC *cli.Context) error {
	ctx := cliC.Context
	configFile := cliC.String("config")

	// nolint:exhaustruct
	cfg := nodeConfig{}
	err := config.LoadFile(&cfg, configFile)
	if err != nil {
		return fmt.Errorf("unable to load config file: %w", err)
	}

	if cfg.NodeID == "" {
		cfg.NodeID = uid.Node()
	}

	if cfg.Region == "" {
		cfg.Region = "unknown"
	}
	logger := logging.New(logging.Config{Development: true, NoColor: false}).
		With(
			slog.String("nodeId", cfg.NodeID),
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

	m, err := membership.New(membership.Config{
		RedisUrl: cfg.RedisUrl,
		NodeID:   cfg.NodeID,
		RpcAddr:  cfg.RpcAddr,
		Logger:   logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create membership")
	}

	c, err := cluster.New(cluster.Config{
		Self: cluster.Node{
			ID:      cfg.NodeID,
			RpcAddr: "",
		},
		Logger:     logger,
		Membership: m,
	})
	if err != nil {
		return fmt.Errorf("unable to create cluster: %w", err)
	}
	_, err = m.Join(ctx)
	if err != nil {
		return fmt.Errorf("unable to join cluster")
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
		NodeID: cfg.NodeID,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)
	}

	validator, err := validation.New()
	if err != nil {
		return fmt.Errorf("unable to create validator: %w", err)
	}

	srv.SetGlobalMiddleware(
		// metrics should always run first, so it can capture the latency of the entire request
		zen.WithMetrics(ch),
		zen.WithLogging(logger),
		zen.WithErrorHandling(),
		zen.WithValidation(validator),
	)

	go func() {
		listenErr := srv.Listen(ctx, cfg.HttpAddr)
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

func init() {
	// nolint:exhaustruct
	_, err := config.GenerateJsonSchema(nodeConfig{}, "schema.json")
	if err != nil {
		panic(err)
	}
}
