package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/config"
	"github.com/unkeyed/unkey/apps/agent/pkg/connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/load"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tinybird"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/services/eventrouter"
	"github.com/unkeyed/unkey/apps/agent/services/vault"
	"github.com/unkeyed/unkey/apps/agent/services/vault/storage"
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

	cfg := configuration{}
	err := config.LoadFile(&cfg, configFile)
	if err != nil {
		return err
	}
	logger, err := setupLogging(cfg)
	if err != nil {
		return err
	}
	if cfg.NodeId == "" {
		cfg.NodeId = uid.Node()
	}
	logger = logger.With().Str("nodeId", cfg.NodeId).Str("region", cfg.Region).Logger()
	logger.Info().Str("file", configFile).Msg("configuration loaded")

	logger.Info().Str("hostname", os.Getenv("HOSTNAME")).Msg("environment")

	tracer := tracing.NewNoop()
	{
		if cfg.Tracing != nil {
			t, closeTracer, err := tracing.New(context.Background(), tracing.Config{
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
			tracer = t
			logger.Info().Msg("tracing to axiom")
		}
	}

	m := metrics.NewNoop()
	if cfg.Metrics != nil && cfg.Metrics.Axiom != nil {
		realMetrics, err := metrics.New(metrics.Config{
			Token:   cfg.Metrics.Axiom.Token,
			Dataset: cfg.Metrics.Axiom.Dataset,
			Logger:  logger.With().Str("pkg", "metrics").Logger(),
			NodeId:  cfg.NodeId,
			Region:  cfg.Region,
		})
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to start metrics")
		}
		m = realMetrics

	}
	defer m.Close()

	l := load.New(load.Config{
		Metrics: m,
		Logger:  logger,
	})
	go l.Start()
	defer l.Stop()

	if cfg.Heartbeat != nil {
		err = setupHeartbeat(cfg, logger)
		if err != nil {
			return err
		}
	}

	srv, err := connect.New(connect.Config{Logger: logger, Tracer: tracer})
	if err != nil {
		return err
	}

	if cfg.Services.Ratelimit != nil {
		rl, err := ratelimit.New(ratelimit.Config{
			Logger: logger,
		})
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create service")
		}
		rl = ratelimit.WithTracing(tracer)(rl)

		srv.AddService(connect.NewRatelimitServer(rl, logger))
		logger.Info().Msg("started ratelimit service")
	}

	if cfg.Services.Vault != nil {
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
		v, err := vault.New(vault.Config{
			Logger:     logger,
			Storage:    s3,
			Metrics:    m,
			MasterKeys: strings.Split(cfg.Services.Vault.MasterKeys, ","),
		})
		if err != nil {
			return fmt.Errorf("failed to create vault: %w", err)
		}

		srv.AddService(connect.NewVaultServer(v, logger))
		logger.Info().Msg("started vault service")
	}

	if cfg.Services.EventRouter != nil {
		er, err := eventrouter.New(eventrouter.Config{
			Logger:        logger,
			Tracer:        tracer,
			BatchSize:     cfg.Services.EventRouter.Tinybird.BatchSize,
			BufferSize:    cfg.Services.EventRouter.Tinybird.BufferSize,
			FlushInterval: time.Duration(cfg.Services.EventRouter.Tinybird.FlushInterval) * time.Second,
			Tinybird:      tinybird.New("https://api.tinybird.co", cfg.Services.EventRouter.Tinybird.Token),
		})
		if err != nil {
			return err
		}
		srv.AddService(er)
	}

	err = srv.Listen(fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return err
	}

	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	<-cShutdown

	return nil
}

type configuration struct {
	NodeId  string `json:"nodeId,omitempty" description:"A unique node id"`
	Logging *struct {
		Axiom *struct {
			Dataset string `json:"dataset" minLength:"1" description:"The dataset to send logs to"`
			Token   string `json:"token" minLength:"1" description:"The token to use for authentication"`
		} `json:"axiom,omitempty" description:"Send logs to axiom"`
	} `json:"logging,omitempty"`

	Tracing *struct {
		Axiom *struct {
			Dataset string `json:"dataset" minLength:"1" description:"The dataset to send traces to"`
			Token   string `json:"token" minLength:"1" description:"The token to use for authentication"`
		} `json:"axiom,omitempty" description:"Send traces to axiom"`
	} `json:"tracing,omitempty"`

	Metrics *struct {
		Axiom *struct {
			Dataset string `json:"dataset" minLength:"1" description:"The dataset to send metrics to"`
			Token   string `json:"token" minLength:"1" description:"The token to use for authentication"`
		} `json:"axiom,omitempty" description:"Send metrics to axiom"`
	} `json:"metrics,omitempty"`

	Schema    string `json:"$schema,omitempty" description:"Make jsonschema happy"`
	Region    string `json:"region,omitempty" description:"The region this agent is running in"`
	Port      int    `json:"port,omitempty" max:"65535" min:"0" default:"8080" description:"Port to listen on"`
	Heartbeat *struct {
		URL      string `json:"url" minLength:"1" description:"URL to send heartbeat to"`
		Interval int    `json:"interval" min:"1" description:"Interval in seconds to send heartbeat"`
	} `json:"heartbeat,omitempty" description:"Send heartbeat to a URL"`

	Services struct {
		Ratelimit *struct {
			AuthToken string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
		} `json:"ratelimit,omitempty" description:"Rate limit requests"`
		EventRouter *struct {
			AuthToken string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
			Tinybird  *struct {
				Token         string `json:"token" minLength:"1" description:"The token to use for tinybird authentication"`
				FlushInterval int    `json:"flushInterval" min:"1" description:"Interval in seconds to flush events"`
				BufferSize    int    `json:"bufferSize" min:"1" description:"Size of the buffer"`
				BatchSize     int    `json:"batchSize" min:"1" description:"Size of the batch"`
			} `json:"tinybird,omitempty" description:"Send events to tinybird"`
		} `json:"eventRouter,omitempty" description:"Route events"`
		Vault *struct {
			S3Bucket          string `json:"s3Bucket" minLength:"1" description:"The bucket to store secrets in"`
			S3Url             string `json:"s3Url" minLength:"1" description:"The url to store secrets in"`
			S3AccessKeyId     string `json:"s3AccessKeyId" minLength:"1" description:"The access key id to use for s3"`
			S3AccessKeySecret string `json:"s3AccessKeySecret" minLength:"1" description:"The access key secret to use for s3"`
			MasterKeys        string `json:"masterKeys" minLength:"1" description:"The master keys to use for encryption, comma separated"`
		} `json:"vault,omitempty" description:"Store secrets"`
	} `json:"services"`
}

// TODO: generating this every time is a bit stupid, we should make this its own command
//
//	and then run it as part of the build process
func init() {
	_, err := config.GenerateJsonSchema(configuration{}, "schema.json")
	if err != nil {
		panic(err)
	}
}
