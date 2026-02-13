package sentinel

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/sentinel"
)

// Cmd is the sentinel command that runs the deployment proxy server for routing
// requests to deployment instances within a specific environment.
var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "sentinel",
	Usage:       "Run the Unkey Sentinel server (deployment proxy)",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the Sentinel server to listen on. Default: 8080",
			cli.Default(8080), cli.EnvVar("UNKEY_HTTP_PORT")),

		// Instance Identification
		cli.String("sentinel-id", "Unique identifier for this sentinel instance. Auto-generated if not provided.",
			cli.Default(uid.New("sentinel", 4)), cli.EnvVar("UNKEY_SENTINEL_ID")),

		cli.String("workspace-id", "Workspace ID this sentinel serves. Required.",
			cli.Required(), cli.EnvVar("UNKEY_WORKSPACE_ID")),

		cli.String("environment-id", "Environment ID this sentinel serves (handles all deployments in this environment). Required.",
			cli.Required(), cli.EnvVar("UNKEY_ENVIRONMENT_ID")),

		cli.String("region", "Geographic region identifier. Used for logging. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION")),

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required.",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),

		cli.String("database-replica", "MySQL connection string for read-replica.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		cli.String("clickhouse-url", "ClickHouse connection string. Optional.",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.", cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		// Gossip Cluster Configuration
		cli.Bool("gossip-enabled", "Enable gossip-based distributed cache invalidation",
			cli.Default(false), cli.EnvVar("UNKEY_GOSSIP_ENABLED")),
		cli.String("gossip-bind-addr", "Address for gossip listeners. Default: 0.0.0.0",
			cli.Default("0.0.0.0"), cli.EnvVar("UNKEY_GOSSIP_BIND_ADDR")),
		cli.Int("gossip-lan-port", "LAN memberlist port. Default: 7946",
			cli.Default(7946), cli.EnvVar("UNKEY_GOSSIP_LAN_PORT")),
		cli.Int("gossip-wan-port", "WAN memberlist port for gateways. Default: 7947",
			cli.Default(7947), cli.EnvVar("UNKEY_GOSSIP_WAN_PORT")),
		cli.StringSlice("gossip-lan-seeds", "LAN seed addresses (e.g. k8s headless service DNS)",
			cli.EnvVar("UNKEY_GOSSIP_LAN_SEEDS")),
		cli.StringSlice("gossip-wan-seeds", "Cross-region gateway seed addresses",
			cli.EnvVar("UNKEY_GOSSIP_WAN_SEEDS")),
		// Logging Sampler Configuration
		cli.Float("log-sample-rate", "Baseline probability (0.0-1.0) of emitting log events. Default: 1.0",
			cli.Default(1.0), cli.EnvVar("UNKEY_LOG_SAMPLE_RATE")),
		cli.Duration("log-slow-threshold", "Duration threshold for slow event sampling. Default: 1s",
			cli.Default(time.Second), cli.EnvVar("UNKEY_LOG_SLOW_THRESHOLD")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	return sentinel.Run(ctx, sentinel.Config{
		// Instance identification
		SentinelID:    cmd.String("sentinel-id"),
		WorkspaceID:   cmd.String("workspace-id"),
		EnvironmentID: cmd.String("environment-id"),
		Region:        cmd.String("region"),

		// HTTP configuration
		HttpPort: cmd.Int("http-port"),

		// Database configuration
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),
		ClickhouseURL:           cmd.String("clickhouse-url"),

		// Observability
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),
		PrometheusPort:        cmd.Int("prometheus-port"),

		// Gossip cluster configuration
		GossipEnabled:  cmd.Bool("gossip-enabled"),
		GossipBindAddr: cmd.String("gossip-bind-addr"),
		GossipLANPort:  cmd.Int("gossip-lan-port"),
		GossipWANPort:  cmd.Int("gossip-wan-port"),
		GossipLANSeeds: cmd.StringSlice("gossip-lan-seeds"),
		GossipWANSeeds: cmd.StringSlice("gossip-wan-seeds"),

		// Logging sampler configuration
		LogSampleRate:    cmd.Float("log-sample-rate"),
		LogSlowThreshold: cmd.Duration("log-slow-threshold"),
	})
}
