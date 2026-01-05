package sentinel

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/sentinel"
)

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
			cli.Default(uid.New("sentinel", 4)), cli.EnvVar("UNKEY_GATEWAY_ID")),

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

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),
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

		// Observability
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),
		PrometheusPort:        cmd.Int("prometheus-port"),
	})
}
