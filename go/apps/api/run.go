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

	"github.com/unkeyed/unkey/go/apps/api/routes"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/aws/ecs"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

// nolint:gocognit
func Run(ctx context.Context, cfg Config) error {

	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	shutdowns := []shutdown.ShutdownFn{}

	clk := clock.New()

	logger := logging.New(logging.Config{Development: true, NoColor: true}).
		With(
			slog.String("nodeId", cfg.ClusterNodeID),
			slog.String("platform", cfg.Platform),
			slog.String("region", cfg.Region),
			slog.String("version", version.Version),
		)

	// Catch any panics now after we have a logger but before we start the server
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	if cfg.OtelOtlpEndpoint != "" {
		shutdownOtel, grafanaErr := otel.InitGrafana(ctx, otel.Config{
			GrafanaEndpoint: cfg.OtelOtlpEndpoint,
			Application:     "api",
			Version:         version.Version,
		})
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
		shutdowns = append(shutdowns, shutdownOtel...)
	}

	db, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
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
	if cfg.ClickhouseURL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL:    cfg.ClickhouseURL,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
	}

	srv, err := zen.New(zen.Config{
		NodeID: cfg.ClusterNodeID,
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
		Permissions: permissions.New(permissions.Config{
			DB:     db,
			Logger: logger,
		}),
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
	logger.Info("shutting down")

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

func setupCluster(cfg Config, logger logging.Logger) (cluster.Cluster, []shutdown.ShutdownFn, error) {
	shutdowns := []shutdown.ShutdownFn{}
	if !cfg.ClusterEnabled {
		return cluster.NewNoop("", "127.0.0.1"), shutdowns, nil
	}

	var advertiseAddr string
	{
		switch {
		case cfg.ClusterAdvertiseAddrStatic != "":
			{
				advertiseAddr = cfg.ClusterAdvertiseAddrStatic
			}
		case cfg.ClusterAdvertiseAddrAwsEcsMetadata:

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

	m, err := membership.New(membership.Config{
		NodeID:        cfg.ClusterNodeID,
		AdvertiseAddr: advertiseAddr,
		GossipPort:    cfg.ClusterGossipPort,
		Logger:        logger,
	})
	if err != nil {
		return nil, shutdowns, fmt.Errorf("unable to create membership: %w", err)
	}

	c, err := cluster.New(cluster.Config{
		Self: cluster.Node{

			ID:      cfg.ClusterNodeID,
			Addr:    advertiseAddr,
			RpcAddr: "TO DO",
		},
		Logger:     logger,
		Membership: m,
		RpcPort:    cfg.ClusterRpcPort,
	})
	if err != nil {
		return nil, shutdowns, fmt.Errorf("unable to create cluster: %w", err)
	}
	shutdowns = append(shutdowns, c.Shutdown)

	var d discovery.Discoverer

	switch {
	case cfg.ClusterDiscoveryStaticAddrs != nil:
		{
			d = &discovery.Static{
				Addrs: cfg.ClusterDiscoveryStaticAddrs,
			}
			break
		}

	case cfg.ClusterDiscoveryRedisURL != "":
		{
			rd, rErr := discovery.NewRedis(discovery.RedisConfig{
				URL:    cfg.ClusterDiscoveryRedisURL,
				NodeID: cfg.ClusterNodeID,
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
