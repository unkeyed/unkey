package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/unkeyed/unkey/apps/agent/pkg/config"
	"github.com/unkeyed/unkey/apps/agent/pkg/connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
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

	logger.Info().Str("file", configFile).Interface("cfg", cfg).Msg("configuration loaded")

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
			defer closeTracer()
			tracer = t
			logger.Info().Msg("tracing to axiom")
		}
	}

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

		rlServer := connect.NewRatelimitServer(rl, logger)

		srv.AddService(rlServer)
		logger.Info().Msg("started ratelimit service")
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

	Schema    string `json:"$schema,omitempty" description:"Make jsonschema happy"`
	Port      int    `json:"port,omitempty" max:"65535" min:"0" default:"8080" description:"Port to listen on"`
	Heartbeat *struct {
		URL      string `json:"url" minLength:"1" description:"URL to send heartbeat to"`
		Interval int    `json:"interval" min:"1" description:"Interval in seconds to send heartbeat"`
	} `json:"heartbeat,omitempty" description:"Send heartbeat to a URL"`

	Services struct {
		Ratelimit *struct {
			AuthToken string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
		} `json:"ratelimit,omitempty" description:"Rate limit requests"`
	}
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
