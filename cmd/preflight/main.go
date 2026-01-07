package preflight

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/preflight"
)

var Cmd = &cli.Command{
	Name:  "preflight",
	Usage: "Run the pod mutation webhook for secrets and credentials injection",
	Flags: []cli.Flag{
		cli.Int("port", "Port for the webhook server",
			cli.Default(8443), cli.EnvVar("WEBHOOK_PORT")),
		cli.String("tls-cert-file", "Path to TLS certificate file",
			cli.Required(), cli.EnvVar("WEBHOOK_TLS_CERT_FILE")),
		cli.String("tls-key-file", "Path to TLS private key file",
			cli.Required(), cli.EnvVar("WEBHOOK_TLS_KEY_FILE")),
		cli.String("unkey-env-image", "Container image for unkey-env binary",
			cli.Default("unkey-env:latest"), cli.EnvVar("UNKEY_ENV_IMAGE")),
		cli.String("unkey-env-image-pull-policy", "Image pull policy (Always, IfNotPresent, Never)",
			cli.Default("IfNotPresent"), cli.EnvVar("UNKEY_ENV_IMAGE_PULL_POLICY")),
		cli.String("krane-endpoint", "Endpoint for Krane secrets service",
			cli.Default("http://krane.unkey.svc.cluster.local:8080"), cli.EnvVar("KRANE_ENDPOINT")),
		cli.String("depot-token", "Depot API token for fetching on-demand pull tokens",
			cli.EnvVar("UNKEY_DEPOT_TOKEN"), cli.Required()),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	config := preflight.Config{
		HttpPort:                cmd.Int("port"),
		TLSCertFile:             cmd.String("tls-cert-file"),
		TLSKeyFile:              cmd.String("tls-key-file"),
		UnkeyEnvImage:           cmd.String("unkey-env-image"),
		UnkeyEnvImagePullPolicy: cmd.String("unkey-env-image-pull-policy"),
		KraneEndpoint:           cmd.String("krane-endpoint"),
		DepotToken:              cmd.String("depot-token"),
	}

	if err := config.Validate(); err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	return preflight.Run(ctx, config)
}
