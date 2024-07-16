package agent

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/config"
	"github.com/unkeyed/unkey/apps/agent/pkg/connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/load"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/tinybird"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/version"
	"github.com/unkeyed/unkey/apps/agent/services/eventrouter"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
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

	cfg := configuration{}
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

	{
		if cfg.Tracing != nil && cfg.Tracing.Axiom != nil {
			closeTracer, err := tracing.Init(context.Background(), tracing.Config{
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
		setupHeartbeat(cfg, logger)

	}

	srv, err := connect.New(connect.Config{Logger: logger, Image: cfg.Image})
	if err != nil {
		return err
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
		s3 = storageMiddleware.WithTracing("s3", s3)
		v, err := vault.New(vault.Config{
			Logger:     logger,
			Storage:    s3,
			Metrics:    m,
			MasterKeys: strings.Split(cfg.Services.Vault.MasterKeys, ","),
		})
		if err != nil {
			return fmt.Errorf("failed to create vault: %w", err)
		}

		err = srv.AddService(connect.NewVaultServer(v, logger, cfg.Services.Vault.AuthToken))
		if err != nil {
			return fmt.Errorf("failed to add vault service: %w", err)
		}
		logger.Info().Msg("started vault service")
	}

	if cfg.Services.EventRouter != nil {
		er, err := eventrouter.New(eventrouter.Config{
			Logger:        logger,
			BatchSize:     cfg.Services.EventRouter.Tinybird.BatchSize,
			BufferSize:    cfg.Services.EventRouter.Tinybird.BufferSize,
			FlushInterval: time.Duration(cfg.Services.EventRouter.Tinybird.FlushInterval) * time.Second,
			Tinybird:      tinybird.New("https://api.tinybird.co", cfg.Services.EventRouter.Tinybird.Token),
			AuthToken:     cfg.Services.EventRouter.AuthToken,
		})
		if err != nil {
			return err
		}
		err = srv.AddService(er)
		if err != nil {
			return fmt.Errorf("failed to add event router service: %w", err)

		}
	}

	var clus cluster.Cluster

	if cfg.Cluster != nil {

		memb, err := membership.New(membership.Config{
			NodeId:   cfg.NodeId,
			RpcAddr:  cfg.Cluster.RpcAddr,
			SerfAddr: cfg.Cluster.SerfAddr,
			Logger:   logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create membership: %w", err)
		}

		var join []string
		if cfg.Cluster.Join.Dns != nil {
			addrs, err := net.LookupHost(cfg.Cluster.Join.Dns.AAAA)
			if err != nil {
				return fmt.Errorf("failed to lookup dns: %w", err)
			}
			logger.Info().Strs("addrs", addrs).Msg("found dns records")
			join = addrs
		} else if cfg.Cluster.Join.Env != nil {
			join = cfg.Cluster.Join.Env.Addrs
		}

		_, err = memb.Join(join...)
		if err != nil {
			return fault.Wrap(err, fmsg.With("failed to join cluster"))
		}
		defer func() {
			logger.Info().Msg("leaving membership")
			err = memb.Leave()
			if err != nil {
				logger.Error().Err(err).Msg("failed to leave cluster")
			}
		}()

		clus, err = cluster.New(cluster.Config{
			NodeId:     cfg.NodeId,
			RpcAddr:    cfg.Cluster.RpcAddr,
			Membership: memb,
			Logger:     logger,
			Debug:      true,
			AuthToken:  cfg.Cluster.AuthToken,
		})
		if err != nil {
			return fmt.Errorf("failed to create cluster: %w", err)
		}
		defer func() {
			err := clus.Shutdown()
			if err != nil {
				logger.Error().Err(err).Msg("failed to shutdown cluster")
			}
		}()

		err = srv.AddService(connect.NewClusterServer(clus, logger))
		if err != nil {
			return fmt.Errorf("failed to add cluster service: %w", err)

		}
	}

	if cfg.Services.Ratelimit != nil {
		rl, err := ratelimit.New(ratelimit.Config{
			Logger:  logger,
			Cluster: clus,
		})
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create service")
		}
		rl = ratelimit.WithTracing(rl)

		err = srv.AddService(connect.NewRatelimitServer(rl, logger, cfg.Services.Ratelimit.AuthToken))
		if err != nil {
			return fmt.Errorf("failed to add ratelimit service: %w", err)
		}
		logger.Info().Msg("started ratelimit service")
	}

	if cfg.Pprof != nil {
		srv.EnablePprof(cfg.Pprof.Username, cfg.Pprof.Password)
	}

	err = srv.Listen(fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		return err
	}

	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	<-cShutdown
	logger.Info().Msg("shutting down")
	err = clus.Shutdown()
	if err != nil {
		return fmt.Errorf("failed to shutdown cluster: %w", err)
	}
	return nil
}

type configuration struct {
	Platform string `json:"platform,omitempty" description:"The platform this agent is running on"`
	NodeId   string `json:"nodeId,omitempty" description:"A unique node id"`
	Image    string `json:"image,omitempty" description:"The image this agent is running"`
	Pprof    *struct {
		Username string `json:"username,omitempty" description:"The username to use for pprof"`
		Password string `json:"password,omitempty" description:"The password to use for pprof"`
	} `json:"pprof,omitempty" description:"Enable pprof"`
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
	Port      string `json:"port,omitempty" default:"8080" description:"Port to listen on"`
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
			AuthToken         string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
		} `json:"vault,omitempty" description:"Store secrets"`
	} `json:"services"`

	Cluster *struct {
		AuthToken string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
		SerfAddr  string `json:"serfAddr" minLength:"1" description:"The host and port for serf to listen on"`
		RpcAddr   string `json:"rpcAddr" minLength:"1" description:"This node's internal address, including protocol and port"`

		Join *struct {
			Env *struct {
				Addrs []string `json:"addrs" description:"Addresses to join, comma separated"`
			} `json:"env,omitempty"`
			Dns *struct {
				AAAA string `json:"aaaa" description:"The AAAA record that returns a comma separated list, containing the ipv6 addresses of all nodes"`
			} `json:"dns,omitempty"`
		} `json:"join,omitempty" description:"The strategy to use to join the cluster"`
	} `json:"cluster,omitempty"`
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
