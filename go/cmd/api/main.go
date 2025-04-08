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
		// Server Configuration
		&cli.IntFlag{
			Name:     "http-port",
			Usage:    "HTTP port for the API server to listen on. Default: 7070",
			Sources:  cli.EnvVars("UNKEY_HTTP_PORT"),
			Value:    7070,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "color",
			Usage:    "Enable colored log output. Default: true",
			Sources:  cli.EnvVars("UNKEY_LOGS_COLOR"),
			Value:    true,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "test-mode",
			Usage:    "Enable test mode. WARNING: Potentially unsafe, may trust client inputs blindly. Default: false",
			Sources:  cli.EnvVars("UNKEY_TEST_MODE"),
			Value:    false,
			Required: false,
		},

		// Instance Identification
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "Cloud platform identifier for this node. Used for logging and metrics.",
			Sources:  cli.EnvVars("UNKEY_PLATFORM"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "image",
			Usage:    "Container image identifier. Used for logging and metrics.",
			Sources:  cli.EnvVars("UNKEY_IMAGE"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "region",
			Usage:    "Geographic region identifier. Used for logging and routing. Default: unknown",
			Sources:  cli.EnvVars("UNKEY_REGION", "AWS_REGION"),
			Value:    "unknown",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "instance-id",
			Usage:    "Unique identifier for this instance. Auto-generated if not provided.",
			Sources:  cli.EnvVars("UNKEY_INSTANCE_ID"),
			Value:    uid.New(uid.InstancePrefix),
			Required: false,
		},

		// Database Configuration
		&cli.StringFlag{
			Name:     "database-primary",
			Usage:    "MySQL connection string for primary database. Required for all deployments. Example: mysql://user:pass@host:3306/unkey?parseTime=true",
			Sources:  cli.EnvVars("UNKEY_DATABASE_PRIMARY_DSN"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     "database-readonly-replica",
			Usage:    "MySQL connection string for read-replica. Reduces load on primary database. Format same as database-primary.",
			Sources:  cli.EnvVars("UNKEY_DATABASE_READONLY_DSN"),
			Required: false,
		},

		// Caching and Storage
		&cli.StringFlag{
			Name:     "redis-url",
			Usage:    "Redis connection string for rate-limiting and distributed counters. Example: redis://localhost:6379",
			Sources:  cli.EnvVars("UNKEY_REDIS_URL"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "clickhouse-url",
			Usage:    "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			Sources:  cli.EnvVars("UNKEY_CLICKHOUSE_URL"),
			Required: false,
		},

		// Observability
		&cli.BoolFlag{
			Name:     "otel",
			Usage:    "Enable OpenTelemetry tracing and metrics. Uses standard OTEL_* environment variables for configuration.",
			Sources:  cli.EnvVars("UNKEY_OTEL"),
			Required: false,
		},
		&cli.FloatFlag{
			Name:     "otel-trace-sampling-rate",
			Usage:    "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel=true. Default: 0.25",
			Sources:  cli.EnvVars("UNKEY_OTEL_TRACE_SAMPLING_RATE"),
			Value:    0.25,
			Required: false,
		},
		&cli.IntFlag{
			Name:     "prometheus-port",
			Usage:    "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			Sources:  cli.EnvVars("UNKEY_PROMETHEUS_PORT"),
			Value:    0,
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
