package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/unkeyed/unkey/apps/agent/pkg/api"
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse"
	"github.com/unkeyed/unkey/apps/agent/pkg/config"
	"github.com/unkeyed/unkey/apps/agent/pkg/connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/profiling"
	"github.com/unkeyed/unkey/apps/agent/pkg/prometheus"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/version"
	"github.com/unkeyed/unkey/apps/agent/services/vault"
	"github.com/unkeyed/unkey/apps/agent/services/vault/storage"
	storageMiddleware "github.com/unkeyed/unkey/apps/agent/services/vault/storage/middleware"
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name: "agent",
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

	cfg := config.Agent{}
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
	logger, err := setupLogging(cfg)
	if err != nil {
		return err
	}
	logger = logger.With().Str("nodeId", cfg.NodeId).Str("platform", cfg.Platform).Str("region", cfg.Region).Str("version", version.Version).Logger()

	// Catch any panics now after we have a logger but before we start the server
	defer func() {
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Bytes("stack", debug.Stack()).Msg("panic")
		}
	}()

	logger.Info().Str("file", configFile).Msg("configuration loaded")

	err = profiling.Start(cfg, logger)
	if err != nil {
		return err
	}

	{
		if cfg.Tracing != nil && cfg.Tracing.Axiom != nil {
			var closeTracer tracing.Closer
			closeTracer, err = tracing.Init(context.Background(), tracing.Config{
				Dataset:     cfg.Tracing.Axiom.Dataset,
				Application: "agent",
				Version:     "1.0.0",
				AxiomToken:  cfg.Tracing.Axiom.Token,
			})
			if err != nil {
				return err
			}
			defer func() {
				err = closeTracer()
				if err != nil {
					logger.Error().Err(err).Msg("failed to close tracer")
				}
			}()
			logger.Info().Msg("tracing to axiom")
		}
	}

	m := metrics.NewNoop()
	if cfg.Metrics != nil && cfg.Metrics.Axiom != nil {
		m, err = metrics.New(metrics.Config{
			Token:   cfg.Metrics.Axiom.Token,
			Dataset: cfg.Metrics.Axiom.Dataset,
			Logger:  logger.With().Str("pkg", "metrics").Logger(),
			NodeId:  cfg.NodeId,
			Region:  cfg.Region,
		})
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to start metrics")
		}

	}
	defer m.Close()

	if cfg.Heartbeat != nil {
		setupHeartbeat(cfg, logger)
	}

	var ch clickhouse.Bufferer = clickhouse.NewNoop()
	if cfg.Clickhouse != nil {
		ch, err = clickhouse.New(clickhouse.Config{
			URL:    cfg.Clickhouse.Url,
			Logger: logger.With().Str("pkg", "clickhouse").Logger(),
		})
		if err != nil {
			return err
		}
	}

	s3, err := storage.NewS3(storage.S3Config{
		S3URL:             cfg.Services.Vault.S3Url,
		S3Bucket:          cfg.Services.Vault.S3Bucket,
		S3AccessKeyId:     cfg.Services.Vault.S3AccessKeyId,
		S3AccessKeySecret: cfg.Services.Vault.S3AccessKeySecret,
		Logger:            logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create s3 storage: %w", err)
	}
	s3 = storageMiddleware.WithTracing("s3", s3)
	v, err := vault.New(vault.Config{
		Logger:     logger,
		Metrics:    m,
		Storage:    s3,
		MasterKeys: strings.Split(cfg.Services.Vault.MasterKeys, ","),
	})
	if err != nil {
		return fmt.Errorf("failed to create vault: %w", err)
	}

	if err != nil {
		return fmt.Errorf("failed to create vault service: %w", err)
	}

	srv, err := api.New(api.Config{
		NodeId:     cfg.NodeId,
		Logger:     logger,
		Ratelimit:  nil,
		Metrics:    m,
		Clickhouse: ch,
		AuthToken:  cfg.AuthToken,
		Vault:      v,
	})
	if err != nil {
		return err
	}

	connectSrv, err := connect.New(connect.Config{Logger: logger, Image: cfg.Image, Metrics: m})
	if err != nil {
		return err
	}

	go func() {
		err = connectSrv.Listen(fmt.Sprintf(":%s", cfg.RpcPort))
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to start connect service")
		}
	}()

	go func() {
		logger.Info().Msgf("listening on port %s", cfg.Port)
		err = srv.Listen(fmt.Sprintf(":%s", cfg.Port))
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to start service")
		}
	}()

	if cfg.Prometheus != nil {
		go func() {
			err = prometheus.Listen(cfg.Prometheus.Path, cfg.Prometheus.Port)
			if err != nil {
				logger.Fatal().Err(err).Msg("failed to start prometheus")
			}
		}()
	}

	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	<-cShutdown
	logger.Info().Msg("shutting down")

	err = connectSrv.Shutdown()
	if err != nil {
		return fmt.Errorf("failed to shutdown connect service: %w", err)
	}
	err = srv.Shutdown()
	if err != nil {
		return fmt.Errorf("failed to shutdown service: %w", err)
	}

	return nil
}

// TODO: generating this every time is a bit stupid, we should make this its own command
//
//	and then run it as part of the build process
func init() {
	_, err := config.GenerateJsonSchema(config.Agent{}, "schema.json")
	if err != nil {
		panic(err)
	}
}
