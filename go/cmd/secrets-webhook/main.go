package secretswebhook

import (
	"context"

	webhook "github.com/unkeyed/unkey/go/apps/secrets-webhook"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Version:  "",
	Aliases:  []string{},
	Commands: []*cli.Command{},
	Name:     "secrets-webhook",
	Usage:    "Run the secrets injection webhook",
	Description: `secrets-webhook is a Kubernetes mutating admission webhook that injects
the unkey-env binary into pods to provide secure secrets injection.

The webhook intercepts pod creation requests and, for pods with the
appropriate labels, modifies them to:
1. Add an init container that copies the unkey-env binary
2. Mount a shared volume for the binary
3. Rewrite container commands to use unkey-env as the entrypoint
4. Add environment variables for secrets configuration

EXAMPLES:
unkey run secrets-webhook --tls-cert-file=/certs/tls.crt --tls-key-file=/certs/tls.key`,
	Flags: []cli.Flag{
		cli.Int("port", "Port for the webhook server to listen on",
			cli.Default(8443), cli.EnvVar("WEBHOOK_PORT")),

		cli.String("tls-cert-file", "Path to TLS certificate file (required)",
			cli.Required(), cli.EnvVar("WEBHOOK_TLS_CERT_FILE")),

		cli.String("tls-key-file", "Path to TLS private key file (required)",
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
