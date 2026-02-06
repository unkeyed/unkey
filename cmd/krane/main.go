package krane

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane"
)

// Cmd is the krane command that runs the Kubernetes deployment service for managing
// container lifecycles and deployments in a Kubernetes cluster.
var Cmd = &cli.Command{
	Version:  "",
	Aliases:  []string{},
	Commands: []*cli.Command{},
	Name:     "krane",
	Usage:    "Run the k8s management service",
	Description: `krane (/kreɪn/) is the kubernetes deployment service for Unkey infrastructure.
It manages the lifecycle of deployments in a kubernetes cluster:

EXAMPLES:
unkey run krane                                   # Run with default configuration`,
	Flags: []cli.Flag{
		// Server Configuration
		cli.String("control-plane-url",
			"URL of the control plane to connect to",
			cli.Default("https://control.unkey.cloud"),
			cli.EnvVar("UNKEY_CONTROL_PLANE_URL"),
		),
		cli.String("control-plane-bearer",
			"Bearer token for authenticating with the control plane",
			cli.Default(""),
			cli.EnvVar("UNKEY_CONTROL_PLANE_BEARER"),
		),

		// Instance Identification
		cli.String("instance-id",
			"Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)),
			cli.EnvVar("UNKEY_INSTANCE_ID"),
		),
		cli.String("region",
			"The cloud region with platform, e.g. us-east-1.aws",
			cli.Required(),
			cli.EnvVar("UNKEY_REGION"),
		),

		cli.String("registry-url",
			"URL of the container registry for pulling images. Example: registry.depot.dev",
			cli.EnvVar("UNKEY_REGISTRY_URL"),
		),

		cli.String("registry-username",
			"Username for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_USERNAME"),
		),

		cli.String("registry-password",
			"Password/token for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_PASSWORD"),
		),

		cli.Int("prometheus-port",
			"Port for Prometheus metrics, set to 0 to disable.",
			cli.Default(0),
			cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		cli.Int("rpc-port",
			"Port for RPC server",
			cli.Default(8070),
			cli.EnvVar("UNKEY_RPC_PORT")),

		// Vault Configuration
		cli.String("vault-url", "URL of the vault service",
			cli.EnvVar("UNKEY_VAULT_URL")),
		cli.String("vault-token", "Authentication token for the vault service",
			cli.EnvVar("UNKEY_VAULT_TOKEN")),

		cli.String("cluster-id", "ID of the cluster",
			cli.Default("local"),
			cli.EnvVar("UNKEY_CLUSTER_ID")),

		// Observability
		cli.Bool("otel-enabled", "Enable OpenTelemetry tracing and logging",
			cli.Default(false),
			cli.EnvVar("UNKEY_OTEL_ENABLED")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for traces (0.0 to 1.0)",
			cli.Default(0.01),
			cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),

		// Logging Sampler Configuration
		cli.Float("log-sample-rate", "Baseline probability (0.0-1.0) of emitting log events. Default: 1.0",
			cli.Default(1.0), cli.EnvVar("UNKEY_LOG_SAMPLE_RATE")),
		cli.Float("log-error-sample-rate", "Probability (0.0-1.0) of emitting log events with errors. Default: 1.0",
			cli.Default(1.0), cli.EnvVar("UNKEY_LOG_ERROR_SAMPLE_RATE")),
		cli.Float("log-slow-sample-rate", "Probability (0.0-1.0) of emitting slow log events. Default: 1.0",
			cli.Default(1.0), cli.EnvVar("UNKEY_LOG_SLOW_SAMPLE_RATE")),
		cli.Duration("log-slow-threshold", "Duration threshold for slow event sampling. Default: 1s",
			cli.Default(time.Second), cli.EnvVar("UNKEY_LOG_SLOW_THRESHOLD")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {

	config := krane.Config{
		Clock:                 nil,
		Region:                cmd.RequireString("region"),
		InstanceID:            cmd.RequireString("instance-id"),
		RegistryURL:           cmd.RequireString("registry-url"),
		RegistryUsername:      cmd.RequireString("registry-username"),
		RegistryPassword:      cmd.RequireString("registry-password"),
		RPCPort:               cmd.RequireInt("rpc-port"),
		VaultURL:              cmd.String("vault-url"),
		VaultToken:            cmd.String("vault-token"),
		PrometheusPort:        cmd.RequireInt("prometheus-port"),
		ControlPlaneURL:       cmd.RequireString("control-plane-url"),
		ControlPlaneBearer:    cmd.RequireString("control-plane-bearer"),
		OtelEnabled:           cmd.Bool("otel-enabled"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		// Logging sampler configuration
		LogSampleRate:      cmd.Float("log-sample-rate"),
		LogErrorSampleRate: cmd.Float("log-error-sample-rate"),
		LogSlowSampleRate:  cmd.Float("log-slow-sample-rate"),
		LogSlowThreshold:   cmd.Duration("log-slow-threshold"),
	}

	// Validate configuration
	err := config.Validate()
	if err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	// Run krane
	return krane.Run(ctx, config)
}
