package vault

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault"
)

// Cmd is the vault command that runs Unkey's encryption service for secure storage
// and retrieval of sensitive data using S3-backed storage.
var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "vault",
	Usage:       "Run unkey's encryption service",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the control plane server to listen on. Default: 8080",
			cli.Default(8060), cli.EnvVar("UNKEY_HTTP_PORT")),

		// Instance Identification
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)), cli.EnvVar("UNKEY_INSTANCE_ID")),

		cli.String("bearer-token", "Authentication token for API access.",
			cli.Required(),
			cli.EnvVar("UNKEY_BEARER_TOKEN")),

		// Vault Configuration - General secrets (env vars, API keys)
		cli.StringSlice("master-keys", "Vault master keys for encryption (general vault)",
			cli.Required(), cli.EnvVar("UNKEY_MASTER_KEYS")),
		cli.String("s3-url", "S3 endpoint URL for general vault",
			cli.Required(),
			cli.EnvVar("UNKEY_S3_URL")),
		cli.String("s3-bucket", "S3 bucket for general vault (env vars, API keys)",
			cli.Required(),
			cli.EnvVar("UNKEY_S3_BUCKET")),
		cli.String("s3-access-key-id", "S3 access key ID for general vault",
			cli.Required(),
			cli.EnvVar("UNKEY_S3_ACCESS_KEY_ID")),
		cli.String("s3-access-key-secret", "S3 secret access key for general vault",
			cli.Required(),
			cli.EnvVar("UNKEY_S3_ACCESS_KEY_SECRET")),

		// Observability
		cli.Bool("otel-enabled", "Enable OpenTelemetry tracing and logging",
			cli.Default(false),
			cli.EnvVar("UNKEY_OTEL_ENABLED")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for traces (0.0 to 1.0)",
			cli.Default(0.01),
			cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.String("region", "Cloud region identifier",
			cli.EnvVar("UNKEY_REGION")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {

	config := vault.Config{
		// Basic configuration
		HttpPort:          cmd.RequireInt("http-port"),
		InstanceID:        cmd.RequireString("instance-id"),
		S3URL:             cmd.RequireString("s3-url"),
		S3Bucket:          cmd.RequireString("s3-bucket"),
		S3AccessKeyID:     cmd.RequireString("s3-access-key-id"),
		S3AccessKeySecret: cmd.RequireString("s3-access-key-secret"),
		MasterKeys:        cmd.RequireStringSlice("master-keys"),
		BearerToken:       cmd.RequireString("bearer-token"),

		// Observability
		OtelEnabled:           cmd.Bool("otel-enabled"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),
		Region:                cmd.String("region"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return vault.Run(ctx, config)
}
