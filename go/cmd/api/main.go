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

	"github.com/unkeyed/unkey/go/cmd/api/routes"
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
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "api",
	Usage: "Run the Unkey API server for validating and managing API keys",

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "platform",
			Usage: `Identifies the cloud platform where this node is running.
This information is primarily used for logging, metrics, and debugging purposes.

Examples:
  --platform=aws     # When running on Amazon Web Services
  --platform=gcp     # When running on Google Cloud Platform
  --platform=hetzner # When running on Hetzner Cloud
  --platform=docker  # When running in Docker (e.g., local or Docker Compose)`,
			Sources:  cli.EnvVars("UNKEY_PLATFORM"),
			Required: false,
		},
		&cli.StringFlag{
			Name: "image",
			Usage: `Container image identifier including repository and tag.
Used for logging and identifying the running version in containerized environments.
Particularly useful when debugging issues between different versions.

Example:
  --image=unkey/api:v1.2.0
  --image=ghcr.io/unkeyed/unkey/api:latest`,
			Sources:  cli.EnvVars("UNKEY_IMAGE"),
			Required: false,
		},
		&cli.IntFlag{
			Name: "http-port",
			Usage: `HTTP port for the API server to listen on.
This port must be accessible by all clients that will interact with the API.
In containerized environments, ensure this port is properly exposed.
The default port is 7070 if not specified.

Examples:
  --http-port=7070  # Default port
  --http-port=8080  # Common alternative for local development
  --http-port=80    # Standard HTTP port (requires root privileges on Unix systems)`,
			Sources:  cli.EnvVars("UNKEY_HTTP_PORT"),
			Value:    7070,
			Required: false,
		},
		&cli.StringFlag{
			Name: "region",
			Usage: `Geographic region identifier where this node is deployed.
Used for logging, metrics categorization, and can affect routing decisions in multi-region setups.
If not specified, defaults to "unknown".

Examples:
  --region=us-east-1    # AWS US East (N. Virginia)
  --region=eu-west-1    # AWS Europe (Ireland)
  --region=us-central1  # GCP US Central
  --region=dev-local    # For local development environments`,
			Sources:  cli.EnvVars("UNKEY_REGION"),
			Value:    "unknown",
			Required: false,
		},

		// Cluster configuration
		&cli.BoolFlag{
			Name: "cluster",
			Usage: `Enable cluster mode to connect multiple Unkey API nodes together.
When enabled, this node will attempt to form or join a cluster with other Unkey nodes.
Clustering provides high availability, load distribution, and consistent rate limiting across nodes.

For production deployments with multiple instances, set this to true.
For single-node setups (local development, small deployments), leave this disabled.

When clustering is enabled, you must also configure:
1. An address advertisement method (static or AWS ECS metadata)
2. A discovery method (static addresses or Redis)
3. Appropriate ports for RPC and gossip protocols

Examples:
  --cluster=true   # Enable clustering
  --cluster=false  # Disable clustering (default)`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER"),
			Required: false,
		},
		&cli.StringFlag{
			Name: "cluster-node-id",
			Usage: `Unique identifier for this node within the cluster.
Every node in a cluster must have a unique identifier. This ID is used in logs,
metrics, and for node-to-node communication within the cluster.

If not specified, a random UUID with 'node_' prefix will be automatically generated.
For ephemeral nodes (like in auto-scaling groups), automatic generation is appropriate.
For stable deployments, consider setting this to a persistent value tied to the instance.

Examples:
  --cluster-node-id=node_east1_001  # For a node in East region, instance 001
  --cluster-node-id=node_replica2   # For a second replica node
  --cluster-node-id=node_dev_local  # For local development`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER_NODE_ID"),
			Value:    uid.New(uid.NodePrefix),
			Required: false,
		},
		&cli.StringFlag{
			Name: "cluster-advertise-addr-static",
			Usage: `Static IP address or hostname that other nodes can use to connect to this node.
This is required for clustering when not using AWS ECS discovery.
The address must be reachable by all other nodes in the cluster.

For on-premises or static cloud deployments, use a fixed IP address or DNS name.
In Kubernetes environments, this could be the pod's DNS name within the cluster.

Only one advertisement method should be configured - either static or AWS ECS metadata.

Examples:
  --cluster-advertise-addr-static=10.0.1.5             # Direct IP address
  --cluster-advertise-addr-static=node1.unkey.internal # DNS name
  --cluster-advertise-addr-static=unkey-0.unkey-headless.default.svc.cluster.local  # Kubernetes DNS`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER_ADVERTISE_ADDR_STATIC", "HOSTNAME"),
			Required: false,
		},
		&cli.BoolFlag{
			Name: "cluster-advertise-addr-aws-ecs-metadata",
			Usage: `Enable automatic address discovery using AWS ECS container metadata.
When running on AWS ECS, this flag allows the container to automatically determine
its private DNS name from the ECS metadata service. This simplifies cluster configuration
in AWS ECS deployments with dynamic IP assignments.

Only one advertisement method should be configured - either static or AWS ECS metadata.
Do not set cluster-advertise-addr-static if this option is enabled.

This option is specifically designed for AWS ECS and won't work in other environments.

Examples:
  --cluster-advertise-addr-aws-ecs-metadata=true  # Enable AWS ECS metadata-based discovery
  --cluster-advertise-addr-aws-ecs-metadata=false # Disable (default)`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER_ADVERTISE_ADDR_AWS_ECS_METADATA"),
			Required: false,
		},
		&cli.IntFlag{
			Name: "cluster-rpc-port",
			Usage: `Port used for internal RPC communication between cluster nodes.
This port is used for direct node-to-node communication within the cluster for
operations like distributed rate limiting and state synchronization.

The port must be accessible by all other nodes in the cluster and should be
different from the HTTP and gossip ports to avoid conflicts.

In containerized environments, ensure this port is properly exposed between containers.
For security, this port should typically not be exposed to external networks.

Examples:
  --cluster-rpc-port=7071  # Default RPC port
  --cluster-rpc-port=9000  # Alternative port if 7071 is unavailable`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER_RPC_PORT"),
			Value:    7071,
			Required: false,
		},
		&cli.IntFlag{
			Name: "cluster-gossip-port",
			Usage: `Port used for cluster membership and failure detection via gossip protocol.
The gossip protocol is used to maintain cluster membership, detect node failures,
and distribute information about the cluster state.

This port must be accessible by all other nodes in the cluster and should be
different from the HTTP and RPC ports to avoid conflicts.

In containerized environments, ensure this port is properly exposed between containers.
For security, this port should typically not be exposed to external networks.

Examples:
  --cluster-gossip-port=7072  # Default gossip port
  --cluster-gossip-port=9001  # Alternative port if 7072 is unavailable`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER_GOSSIP_PORT"),
			Value:    7072,
			Required: false,
		},
		// Discovery configuration - static
		&cli.StringSliceFlag{
			Name: "cluster-discovery-static-addrs",
			Usage: `List of seed node addresses for static cluster configuration.
When using static discovery, these addresses serve as initial contact points for
joining the cluster. At least one functioning node address must be provided for
initial cluster formation.

This flag is required for clustering when not using Redis discovery.
Each address should be a hostname or IP address that's reachable by this node.
It's not necessary to list all nodes - just enough to ensure reliable discovery.

Nodes will automatically discover the full cluster membership after connecting to
any existing cluster member.

Examples:
  --cluster-discovery-static-addrs=10.0.1.5,10.0.1.6
  --cluster-discovery-static-addrs=node1.unkey.internal,node2.unkey.internal
  --cluster-discovery-static-addrs=unkey-0.unkey-headless.default.svc.cluster.local`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER_DISCOVERY_STATIC_ADDRS"),
			Required: false,
		},
		// Discovery configuration - Redis
		&cli.StringFlag{
			Name: "cluster-discovery-redis-url",
			Usage: `Redis connection string for dynamic cluster discovery.
Redis-based discovery enables nodes to register themselves and discover other nodes
through a shared Redis instance. This is recommended for dynamic environments where
nodes may come and go frequently, such as auto-scaling groups in AWS ECS.

When specified, nodes will register themselves in Redis and discover other nodes
automatically. This eliminates the need for static address configuration.

The Redis instance should be accessible by all nodes in the cluster and have
low latency to ensure timely node discovery.

Examples:
  --cluster-discovery-redis-url=redis://localhost:6379/0
  --cluster-discovery-redis-url=redis://user:password@redis.example.com:6379/0
  --cluster-discovery-redis-url=redis://user:password@redis-master.default.svc.cluster.local:6379/0?tls=true`,
			Sources:  cli.EnvVars("UNKEY_CLUSTER_DISCOVERY_REDIS_URL"),
			Required: false,
		},
		// Logs configuration
		&cli.BoolFlag{
			Name: "color",
			Usage: `Enable ANSI color codes in log output.
When enabled, log output will include ANSI color escape sequences to highlight
different log levels, timestamps, and other components of the log messages.

This is useful for local development and debugging but should typically be disabled
in production environments where logs are collected by systems that may not
properly handle ANSI escape sequences (e.g., CloudWatch, Loki, or other log collectors).

Examples:
  --color=true   # Enable colored logs (good for local development)
  --color=false  # Disable colored logs (default, best for production)`,
			Sources:  cli.EnvVars("UNKEY_LOGS_COLOR"),
			Required: false,
		},
		// Clickhouse configuration
		&cli.StringFlag{
			Name: "clickhouse-url",
			Usage: `ClickHouse database connection string for analytics and audit logs.
ClickHouse is used for storing high-volume event data like API key validations,
creating a complete audit trail of all operations and enabling advanced analytics.

This is optional but highly recommended for production environments. If not provided,
analytical capabilities will be limited but core key validation will still function.

The ClickHouse database should be properly configured for time-series data and
have adequate storage for your expected usage volume.

Examples:
  --clickhouse-url=clickhouse://localhost:9000/unkey
  --clickhouse-url=clickhouse://user:password@clickhouse.example.com:9000/unkey
  --clickhouse-url=clickhouse://default:password@clickhouse.default.svc.cluster.local:9000/unkey?secure=true`,
			Sources:  cli.EnvVars("UNKEY_CLICKHOUSE_URL"),
			Required: false,
		},
		// Database configuration
		&cli.StringFlag{
			Name: "database-primary",
			Usage: `Primary database connection string for read and write operations.
This MySQL database stores all persistent data including API keys, workspaces,
and configuration. It is required for all deployments.

For production use, ensure the database has proper backup procedures in place
and consider using a managed MySQL service with high availability.

The connection string must be a valid MySQL connection string with all
necessary parameters, including SSL mode for secure connections.

Examples:
  --database-primary=mysql://root:password@localhost:3306/unkey
  --database-primary=mysql://user:password@mysql.example.com:3306/unkey?tls=true
  --database-primary=mysql://unkey:password@mysql.default.svc.cluster.local:3306/unkey`,
			Sources:  cli.EnvVars("UNKEY_DATABASE_PRIMARY_DSN"),
			Required: true,
		},
		&cli.StringFlag{
			Name: "database-readonly-replica",
			Usage: `Optional read-replica database connection string for read operations.
When provided, read operations that don't require the latest data will be directed
to this read replica, reducing load on the primary database.

This is recommended for high-traffic deployments to improve performance and scalability.
The read replica must be a valid MySQL read replica of the primary database.

In AWS, this could be an RDS read replica. In other environments, it could be a
MySQL replica configured with binary log replication.

Examples:
  --database-readonly-replica=mysql://readonly:password@replica.mysql.example.com:3306/unkey?tls=true
  --database-readonly-replica=mysql://readonly:password@mysql-replica.default.svc.cluster.local:3306/unkey`,
			Sources:  cli.EnvVars("UNKEY_DATABASE_READONLY_DSN"),
			Required: false,
		},
		// OpenTelemetry configuration
		&cli.StringFlag{
			Name: "otel-otlp-endpoint",
			Usage: `OpenTelemetry collector endpoint for metrics, traces, and logs.
When provided, the Unkey API will send telemetry data (metrics, traces, and logs)
to this endpoint using the OTLP protocol. This enables comprehensive observability
for production deployments.

The endpoint should be an OpenTelemetry collector capable of receiving OTLP data.
The implementation is currently configured for Grafana Cloud integration but is
compatible with any OTLP-compliant collector.

Enabling telemetry is highly recommended for production deployments to monitor
performance, detect issues, and troubleshoot problems.

Examples:
  --otel-otlp-endpoint=http://localhost:4317                    # Local collector
  --otel-otlp-endpoint=https://otlp.grafana-cloud.example.com   # Grafana Cloud
  --otel-otlp-endpoint=https://api.honeycomb.io:443             # Honeycomb.io`,
			Sources:  cli.EnvVars("UNKEY_OTEL_OTLP_ENDPOINT"),
			Required: false,
		},
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg := configFromFlags(cmd)

	return run(ctx, cfg)
}

// nolint:gocognit
func run(ctx context.Context, cfg nodeConfig) error {

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

func setupCluster(cfg nodeConfig, logger logging.Logger) (cluster.Cluster, []shutdown.ShutdownFn, error) {
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
