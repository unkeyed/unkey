package controlplane

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/controlplane"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "controlplane",
	Usage: "Run the Unkey controlplane service for background workflow processing",

	Flags: []cli.Flag{
		// Instance Identification
		&cli.StringFlag{
			Name:     "image",
			Usage:    "Container image identifier. Used for logging and metrics.",
			Sources:  cli.EnvVars("UNKEY_IMAGE"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "instance-id",
			Usage:    "Unique identifier for this instance. Auto-generated if not provided.",
			Sources:  cli.EnvVars("UNKEY_INSTANCE_ID"),
			Value:    uid.New(uid.InstancePrefix, 4),
			Required: false,
		},

		// Database Configuration
		&cli.StringFlag{
			Name:     "hydra-database-dsn",
			Usage:    "Database connection for workflow persistence. MySQL DSN (user:pass@host:3306/unkey?parseTime=true).",
			Sources:  cli.EnvVars("UNKEY_HYDRA_DATABASE_DSN"),
			Required: false,
		},

		// Business Database Configuration (for quota check and other business workflows)
		&cli.StringFlag{
			Name:     "unkey-database-dsn",
			Usage:    "Business database connection for accessing workspace data. Required for quota check workflow. Example: user:pass@host:3306/unkey?parseTime=true",
			Sources:  cli.EnvVars("UNKEY_DATABASE_DSN"),
			Required: true,
		},

		// External Services Configuration
		&cli.StringFlag{
			Name:     "clickhouse-url",
			Usage:    "ClickHouse connection string for analytics data. Required for quota check workflow. Example: clickhouse://user:pass@host:9000/unkey",
			Sources:  cli.EnvVars("UNKEY_CLICKHOUSE_URL"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     "slack-webhook-url",
			Usage:    "Slack webhook URL for quota violation notifications. Optional for quota check workflow.",
			Sources:  cli.EnvVars("UNKEY_SLACK_WEBHOOK_URL"),
			Required: false,
		},

		// Observability
		&cli.BoolFlag{
			Name:     "otel",
			Usage:    "Enable OpenTelemetry tracing and metrics",
			Sources:  cli.EnvVars("UNKEY_OTEL"),
			Required: false,
		},
		&cli.FloatFlag{
			Name:     "otel-trace-sampling-rate",
			Usage:    "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			Sources:  cli.EnvVars("UNKEY_OTEL_TRACE_SAMPLING_RATE"),
			Value:    0.25,
			Required: false,
		},
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {

	config := controlplane.Config{
		// Basic configuration
		Image:      cmd.String("image"),
		InstanceID: cmd.String("instance-id"),

		// Database configuration
		HydraDatabaseDSN: cmd.String("hydra-database-dsn"),

		// Business database configuration
		UnkeyDatabaseDSN: cmd.String("unkey-database-dsn"),

		// External services configuration
		ClickHouseURL:   cmd.String("clickhouse-url"),
		SlackWebhookURL: cmd.String("slack-webhook-url"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		// Clock (real clock for production)
		Clock: clock.New(),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return controlplane.Run(ctx, config)
}
