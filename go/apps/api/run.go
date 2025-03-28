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
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/aws/ecs"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rpc"
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

	clk := clock.New()

	shutdowns := shutdown.New()

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "api",
			Version:         version.Version,
			InstanceID:      cfg.ClusterInstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		},
			shutdowns,
		)
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

	logger := logging.New().
		With(
			slog.String("instanceID", cfg.ClusterInstanceID),
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

	db, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	defer db.Close()

	c, err := setupCluster(cfg, logger, shutdowns)
	if err != nil {
		return fmt.Errorf("unable to create cluster: %w", err)
	}

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickhouseURL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL:    cfg.ClickhouseURL,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
	}

	caches, err := caches.New(caches.Config{
		Logger: logger,
		Clock:  clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	srv, err := zen.New(zen.Config{
		InstanceID: cfg.ClusterInstanceID,
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)

	}
	shutdowns.Register(srv.Shutdown)

	validator, err := validation.New()
	if err != nil {
		return fmt.Errorf("unable to create validator: %w", err)
	}

	keySvc, err := keys.New(keys.Config{
		Logger:   logger,
		DB:       db,
		Clock:    clk,
		KeyCache: caches.KeyByHash,
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

	if cfg.ClusterEnabled {

		rpcSvc, rpcErr := rpc.New(rpc.Config{
			Logger:           logger,
			RatelimitService: rlSvc,
		})
		if rpcErr != nil {
			return fmt.Errorf("unable to create rpc service: %w", rpcErr)
		}

		go func() {
			listenErr := rpcSvc.Listen(ctx, fmt.Sprintf(":%d", cfg.ClusterRpcPort))
			if listenErr != nil {
				panic(listenErr)
			}
		}()
	}

	p, err := permissions.New(permissions.Config{
		DB:     db,
		Logger: logger,
		Clock:  clk,
		Cache:  caches.PermissionsByKeyId,
	})
	if err != nil {
		return fmt.Errorf("unable to create permissions service: %w", err)
	}

	routes.Register(srv, &routes.Services{
		Logger:      logger,
		Database:    db,
		ClickHouse:  ch,
		Keys:        keySvc,
		Validator:   validator,
		Ratelimit:   rlSvc,
		Permissions: p,
		Caches:      caches,
	})

	go func() {
		listenErr := srv.Listen(ctx, fmt.Sprintf(":%d", cfg.HttpPort))
		if listenErr != nil {
			panic(listenErr)
		}
	}()

	return gracefulShutdown(ctx, logger, shutdowns)
}
func gracefulShutdown(ctx context.Context, logger logging.Logger, shutdowns *shutdown.Shutdowns) error {
	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	// Create a channel that closes when the context is done
	done := ctx.Done()

	// Wait for either a signal or context cancellation
	select {
	case <-cShutdown:
		logger.Info("shutting down due to signal")
	case <-done:
		logger.Info("shutting down due to context cancellation")
	}

	// Create a timeout context for the shutdown process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	errs := shutdowns.Shutdown(shutdownCtx)

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred during shutdown: %v", errs)
	}
	return nil
}
func setupCluster(cfg Config, logger logging.Logger, shutdowns *shutdown.Shutdowns) (cluster.Cluster, error) {
	if !cfg.ClusterEnabled {
		return cluster.NewNoop("", "127.0.0.1"), nil
	}

	var advertiseHost string
	{
		switch {
		case cfg.ClusterAdvertiseAddrStatic != "":
			{
				advertiseHost = cfg.ClusterAdvertiseAddrStatic
			}
		case cfg.ClusterAdvertiseAddrAwsEcsMetadata:

			{
				var getDnsErr error
				advertiseHost, getDnsErr = ecs.GetPrivateDnsName()
				if getDnsErr != nil {
					return nil, fmt.Errorf("unable to get private dns name: %w", getDnsErr)
				}

			}
		default:
			return nil, fmt.Errorf("invalid advertise address configuration")
		}
	}

	m, err := membership.New(membership.Config{
		InstanceID:    cfg.ClusterInstanceID,
		AdvertiseHost: advertiseHost,
		GossipPort:    cfg.ClusterGossipPort,
		RpcPort:       cfg.ClusterRpcPort,
		HttpPort:      cfg.HttpPort,
		Logger:        logger,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create membership: %w", err)
	}

	c, err := cluster.New(cluster.Config{
		Self: cluster.Instance{
			ID:      cfg.ClusterInstanceID,
			RpcAddr: fmt.Sprintf("%s:%d", advertiseHost, cfg.ClusterRpcPort),
		},
		Logger:     logger,
		Membership: m,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create cluster: %w", err)
	}
	shutdowns.RegisterCtx(c.Shutdown)

	var d discovery.Discoverer

	if cfg.ClusterDiscoveryRedisURL != "" {
		rd, rErr := discovery.NewRedis(discovery.RedisConfig{
			URL:        cfg.ClusterDiscoveryRedisURL,
			InstanceID: cfg.ClusterInstanceID,
			Addr:       fmt.Sprintf("%s:%d", advertiseHost, cfg.ClusterGossipPort),
			Logger:     logger,
		})
		if rErr != nil {
			return nil, fmt.Errorf("unable to create redis discovery: %w", rErr)
		}
		shutdowns.RegisterCtx(rd.Shutdown)
		d = rd
	} else {
		d = &discovery.Static{
			Addrs: cfg.ClusterDiscoveryStaticAddrs,
		}
	}

	err = m.Start(d)
	if err != nil {
		return nil, fmt.Errorf("unable to start membership: %w", err)
	}

	return c, nil
}
