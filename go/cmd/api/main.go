package api

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/unkeyed/unkey/go/pkg/api/server"
	"github.com/unkeyed/unkey/go/pkg/config"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/version"
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

func run(c *cli.Context) error {
	configFile := c.String("config")

	cfg := nodeConfig{}
	err := config.LoadFile(&cfg, configFile)
	if err != nil {
		return err
	}

	if cfg.NodeId == "" {
		cfg.NodeId = uid.Node()

	}

	if cfg.Region == "" {
		cfg.Region = "unknown"
	}
	logger := logging.New(logging.Config{Development: true})
	logger = logger.With(

		slog.String("nodeId", cfg.NodeId),
		slog.String("platform", cfg.Platform),
		slog.String("region", cfg.Region),
		slog.String("version", version.Version),
	)

	// Catch any panics now after we have a logger but before we start the server
	defer func() {
		if r := recover(); r != nil {
			logger.Error(c.Context, "panic",
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()

	logger.Info(c.Context, "configration loaded", slog.String("file", configFile))

	srv, err := server.New(server.Config{
		NodeId:     cfg.NodeId,
		Logger:     logger,
		Clickhouse: nil,
	})
	if err != nil {
		return err
	}

	go func() {
		listenErr := srv.Listen(c.Context, fmt.Sprintf(":%s", cfg.Port))
		if listenErr != nil {
			panic(listenErr)
		}
	}()

	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	<-cShutdown
	logger.Info(c.Context, "shutting down")

	return nil
}

// TODO: generating this every time is a bit stupid, we should make this its own command
//
//	and then run it as part of the build process
func init() {
	_, err := config.GenerateJsonSchema(nodeConfig{}, "schema.json")
	if err != nil {
		panic(err)
	}
}
