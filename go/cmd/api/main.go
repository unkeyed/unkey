package api

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/uid"
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
  --http-port=7070  # Default port`,
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
			Sources:  cli.EnvVars("UNKEY_REGION", "AWS_REGION"),
			Value:    "unknown",
			Required: false,
		},

		&cli.StringFlag{
			Name:     "instance-id",
			Usage:    "Unique identifier for this instance within the cluster.",
			Sources:  cli.EnvVars("UNKEY_INSTANCE_ID"),
			Value:    uid.New(uid.InstancePrefix),
			Required: false,
		},

		// Redis
		&cli.StringFlag{
			Name:     "redis-url",
			Usage:    "Redis connection string for cross-cluster semi-durable storage of counters.",
			Sources:  cli.EnvVars("UNKEY_REDIS_URL"),
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
	--database-primary=mysql://root:password@localhost:3306/unkey?parseTime=true
  --database-primary=mysql://username:pscale_pw_...@aws.connect.psdb.cloud/unkey?sslmode=require`,
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
	--database-readonly-replica=mysql://root:password@localhost:3306/unkey?parseTime=true
	--database-readonly-replica=mysql://username:pscale_pw_...@aws.connect.psdb.cloud/unkey?sslmode=require`,
			Sources:  cli.EnvVars("UNKEY_DATABASE_READONLY_DSN"),
			Required: false,
		},
		// OpenTelemetry configuration
		&cli.BoolFlag{
			Name: "otel",
			Usage: `Enable OpenTelemetry tracing and metrics.
When enabled, the Unkey API will collect and export telemetry data (metrics, traces, and logs)
using the OpenTelemetry protocol. This provides comprehensive observability for production deployments.

When this flag is set to true, the following standard OpenTelemetry environment variables are used:
- OTEL_EXPORTER_OTLP_ENDPOINT: The URL of your OpenTelemetry collector
- OTEL_EXPORTER_OTLP_PROTOCOL: The protocol to use (http/protobuf or grpc)
- OTEL_EXPORTER_OTLP_HEADERS: Headers for authentication (e.g., "authorization=Bearer <token>")

For more information on these variables, see:
https://grafana.com/docs/grafana-cloud/send-data/otlp/send-data-otlp/

Examples:
  --otel=true   # Enable OpenTelemetry with environment variable configuration
  --otel=false  # Disable OpenTelemetry (default)`,
			Sources:  cli.EnvVars("UNKEY_OTEL"),
			Required: false,
		},
		&cli.FloatFlag{
			Name: "otel-trace-sampling-rate",
			Usage: `Sets the sampling rate for OpenTelemetry traces as a value between 0.0 and 1.0.
This controls what percentage of traces will be collected and exported, helping to balance
observability needs with performance and cost considerations.

- 0.0 means no traces are sampled (0%)
- 0.25 means 25% of traces are sampled (default)
- 1.0 means all traces are sampled (100%)

Lower sampling rates reduce overhead and storage costs but provide less visibility.
Higher rates give more comprehensive data but increase resource usage and costs.

This setting only takes effect when OpenTelemetry is enabled with --otel=true.

Examples:
  --otel-trace-sampling-rate=0.1   # Sample 10% of traces
  --otel-trace-sampling-rate=0.25  # Sample 25% of traces (default)
  --otel-trace-sampling-rate=1.0   # Sample all traces`,
			Sources:  cli.EnvVars("UNKEY_OTEL_TRACE_SAMPLING_RATE"),
			Value:    0.25,
			Required: false,
		},
		&cli.IntFlag{
			Name: "prometheus-port",
			Usage: `Enables prometheus and configures the exposed port.
Metrics will be available at /metrics and http service discovery at /sd.

Default: disabled
			`,
			Sources:  cli.EnvVars("UNKEY_PROMETHEUS_PORT"),
			Value:    0,
			Required: false,
		},
		&cli.BoolFlag{
			Name: "test-mode",
			Usage: `Enable test mode. This is potentially unsafe.
Testmode enables some flags for testing purposes and may trust client inputs blindly.

Default: disabled
			`,
			Sources:  cli.EnvVars("UNKEY_TEST_MODE"),
			Value:    false,
			Required: false,
		},
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {

	config := api.Config{
		// Basic configuration
		Platform: cmd.String("platform"),
		Image:    cmd.String("image"),
		HttpPort: int(cmd.Int("http-port")),
		Region:   cmd.String("region"),

		// Database configuration
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-readonly-replica"),

		// Logs
		LogsColor: cmd.Bool("color"),

		// ClickHouse
		ClickhouseURL: cmd.String("clickhouse-url"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		InstanceID:     cmd.String("instance-id"),
		RedisUrl:       cmd.String("redis-url"),
		PrometheusPort: int(cmd.Int("prometheus-port")),
		Clock:          clock.New(),
		TestMode:       cmd.Bool("test-mode"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return api.Run(ctx, config)
}
