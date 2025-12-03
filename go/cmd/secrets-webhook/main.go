package secretswebhook

import (
	"context"

	webhook "github.com/unkeyed/unkey/go/apps/secrets-webhook"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "secrets-webhook",
	Usage: "Run the secrets injection webhook",
	Flags: []cli.Flag{
		cli.Int("port", "Port for the webhook server",
			cli.Default(8443), cli.EnvVar("WEBHOOK_PORT")),
		cli.String("tls-cert-file", "Path to TLS certificate file",
			cli.Required(), cli.EnvVar("WEBHOOK_TLS_CERT_FILE")),
		cli.String("tls-key-file", "Path to TLS private key file",
			cli.Required(), cli.EnvVar("WEBHOOK_TLS_KEY_FILE")),
		cli.String("unkey-env-image", "Container image for unkey-env binary",
			cli.Default("unkey-env:latest"), cli.EnvVar("UNKEY_ENV_IMAGE")),
		cli.String("krane-endpoint", "Endpoint for Krane secrets service",
			cli.Default("http://krane.unkey.svc.cluster.local:8080"), cli.EnvVar("KRANE_ENDPOINT")),
		cli.String("annotation-prefix", "Annotation prefix for pod configuration",
			cli.Default("unkey.com"), cli.EnvVar("ANNOTATION_PREFIX")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	config := webhook.Config{
		HttpPort:         cmd.Int("port"),
		TLSCertFile:      cmd.String("tls-cert-file"),
		TLSKeyFile:       cmd.String("tls-key-file"),
		UnkeyEnvImage:    cmd.String("unkey-env-image"),
		KraneEndpoint:    cmd.String("krane-endpoint"),
		AnnotationPrefix: cmd.String("annotation-prefix"),
	}

	if err := config.Validate(); err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	return webhook.Run(ctx, config)
}
