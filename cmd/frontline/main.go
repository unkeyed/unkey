package frontline

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/frontline"
)

// Cmd is the frontline command that runs the multi-tenant ingress server for TLS
// termination and request routing to backend services.
var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "frontline",
	Usage:       "Run the Unkey Frontline server (multi-tenant frontline)",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the Gate server to listen on. Default: 7070",
			cli.Default(7070), cli.EnvVar("UNKEY_HTTP_PORT")),

		cli.Int("https-port", "HTTPS port for the Gate server to listen on. Default: 7443",
			cli.Default(7443), cli.EnvVar("UNKEY_HTTPS_PORT")),

		cli.Bool("tls-enabled", "Enable TLS termination for the frontline. Default: true",
			cli.Default(true), cli.EnvVar("UNKEY_TLS_ENABLED")),

		cli.String("tls-cert-file", "Path to TLS certificate file (dev mode)",
			cli.EnvVar("UNKEY_TLS_CERT_FILE")),

		cli.String("tls-key-file", "Path to TLS key file (dev mode)",
			cli.EnvVar("UNKEY_TLS_KEY_FILE")),

		cli.String("region", "The cloud region with platform, e.g. us-east-1.aws",
			cli.Required(),
			cli.EnvVar("UNKEY_REGION"),
		),

		cli.String("frontline-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New("frontline", 4)), cli.EnvVar("UNKEY_GATE_ID")),

		cli.String("default-cert-domain", "Domain to use for fallback TLS certificate when a domain has no cert configured",
			cli.EnvVar("UNKEY_DEFAULT_CERT_DOMAIN")),

		cli.String("apex-domain", "Apex domain for region routing. Cross-region requests forwarded to frontline.{region}.{apex-domain}. Example: unkey.cloud",
			cli.Default("unkey.cloud"), cli.EnvVar("UNKEY_APEX_DOMAIN")),

		// Database Configuration - Partitioned (for hostname lookups)
		cli.String("database-primary", "MySQL connection string for partitioned primary database (frontline operations). Required. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),

		cli.String("database-replica", "MySQL connection string for partitioned read-replica (frontline operations). Format same as database-primary.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.", cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		// Vault Configuration
		cli.String("vault-url", "URL of the remote vault service (e.g., http://vault:8080)",
			cli.EnvVar("UNKEY_VAULT_URL")),
		cli.String("vault-token", "Authentication token for the vault service",
			cli.EnvVar("UNKEY_VAULT_TOKEN")),

		cli.Int("max-hops", "Maximum number of hops allowed for a request",
			cli.Default(10), cli.EnvVar("UNKEY_MAX_HOPS")),

		cli.String("ctrl-addr", "Address of the control plane",
			cli.Default("localhost:8080"), cli.EnvVar("UNKEY_CTRL_ADDR")),

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
	config := frontline.Config{
		// Basic configuration
		FrontlineID: cmd.String("frontline-id"),
		Image:       cmd.String("image"),
		Region:      cmd.String("region"),

		// HTTP configuration
		HttpPort:  cmd.Int("http-port"),
		HttpsPort: cmd.Int("https-port"),

		// TLS configuration
		EnableTLS:   cmd.Bool("tls-enabled"),
		TLSCertFile: cmd.String("tls-cert-file"),
		TLSKeyFile:  cmd.String("tls-key-file"),
		ApexDomain:  cmd.String("apex-domain"),
		MaxHops:     cmd.Int("max-hops"),

		// Control Plane Configuration
		CtrlAddr: cmd.String("ctrl-addr"),

		// Partitioned Database configuration (for hostname lookups)
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),
		PrometheusPort:        cmd.Int("prometheus-port"),

		// Vault configuration
		VaultURL:   cmd.String("vault-url"),
		VaultToken: cmd.String("vault-token"),

		// Logging sampler configuration
		LogSampleRate:    cmd.Float("log-sample-rate"),
		LogSlowThreshold: cmd.Duration("log-slow-threshold"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return frontline.Run(ctx, config)
}
