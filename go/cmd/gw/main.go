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

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required for all deployments. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("database-replica", "MySQL connection string for read-replica. Reduces load on primary database. Format same as database-primary.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		// ClickHouse Proxy Service Configuration
		cli.String(
			"chproxy-auth-token",
			"Authentication token for ClickHouse proxy endpoints. Required when proxy is enabled.",
			cli.EnvVar("UNKEY_CHPROXY_AUTH_TOKEN"),
		),
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {

	config := gw.Config{
		// Basic configuration
		Platform: cmd.String("platform"),
		Image:    cmd.String("image"),
		Region:   cmd.String("region"),

		// Database configuration
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		// ClickHouse
		ClickhouseURL: cmd.String("clickhouse-url"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		GatewayID:      cmd.String("gateway-id"),
		PrometheusPort: cmd.Int("prometheus-port"),

		// HTTP configuration
		HttpPort: cmd.Int("http-port"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return gw.Run(ctx, config)
}
