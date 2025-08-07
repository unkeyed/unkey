package gw

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/gw"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var Cmd = &cli.Command{
	Name:  "gw",
	Usage: "Run the Unkey Gateway server",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the API server to listen on. Default: 6060",
			cli.Default(6060), cli.EnvVar("UNKEY_HTTP_PORT")),

		// Instance Identification
		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics.",
			cli.EnvVar("UNKEY_PLATFORM")),
		cli.String("image", "Container image identifier. Used for logging and metrics.",
			cli.EnvVar("UNKEY_IMAGE")),
		cli.String("region", "Geographic region identifier. Used for logging and routing. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION"), cli.EnvVar("AWS_REGION")),
		cli.String("gateway-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.GatewayPrefix, 4)), cli.EnvVar("UNKEY_GATEWAY_ID")),
		cli.Bool("tls-enabled", "Enable TLS termination for the gateway. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_TLS_ENABLED")),

		// Database Configuration - Partitioned (for gateway operations)
		cli.String("database-primary", "MySQL connection string for partitioned primary database (gateway operations). Required. Example: user:pass@host:3306/partition_001?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("database-replica", "MySQL connection string for partitioned read-replica (gateway operations). Format same as database-primary.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		// Database Configuration - Keys Service
		cli.String("keys-database-primary", "MySQL connection string for keys service primary database (non-partitioned). Required. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_KEYS_DATABASE_PRIMARY")),
		cli.String("keys-database-replica", "MySQL connection string for keys service read-replica (non-partitioned). Format same as keys-database-primary.",
			cli.EnvVar("UNKEY_KEYS_DATABASE_REPLICA")),

		// ClickHouse Configuration
		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	config := gw.Config{
		// Basic configuration
		GatewayID: cmd.String("gateway-id"),
		Platform:  cmd.String("platform"),
		Image:     cmd.String("image"),
		Region:    cmd.String("region"),

		// HTTP configuration
		HttpPort: cmd.Int("http-port"),

		// TLS configuration
		EnableTLS: cmd.Bool("tls-enabled"),

		// Partitioned Database configuration (for gateway operations)
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		// Keys Database configuration
		KeysDatabasePrimary:         cmd.String("keys-database-primary"),
		KeysDatabaseReadonlyReplica: cmd.String("keys-database-replica"),

		// ClickHouse
		ClickhouseURL: cmd.String("clickhouse-url"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),
		PrometheusPort:        cmd.Int("prometheus-port"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return gw.Run(ctx, config)
}
