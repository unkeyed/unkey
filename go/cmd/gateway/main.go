package gateway

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/gateway"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "gateway",
	Usage:       "Run the Unkey Gateway server (deployment proxy)",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the Gateway server to listen on. Default: 8080",
			cli.Default(8080), cli.EnvVar("UNKEY_HTTP_PORT")),

		// Instance Identification
		cli.String("gateway-id", "Unique identifier for this gateway instance. Auto-generated if not provided.",
			cli.Default(uid.New("gateway", 4)), cli.EnvVar("UNKEY_GATEWAY_ID")),

		cli.String("workspace-id", "Workspace ID this gateway serves. Required.",
			cli.Required(), cli.EnvVar("UNKEY_WORKSPACE_ID")),

		cli.String("environment-id", "Environment ID this gateway serves (handles all deployments in this environment). Required.",
			cli.Required(), cli.EnvVar("UNKEY_ENVIRONMENT_ID")),

		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics.",
			cli.EnvVar("UNKEY_PLATFORM")),

		cli.String("region", "Geographic region identifier. Used for logging. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION"), cli.EnvVar("AWS_REGION")),

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
	return gateway.Run(ctx, gateway.Config{
		// Instance identification
		GatewayID:     cmd.String("gateway-id"),
		WorkspaceID:   cmd.String("workspace-id"),
		EnvironmentID: cmd.String("environment-id"),
		Platform:      cmd.String("platform"),
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
