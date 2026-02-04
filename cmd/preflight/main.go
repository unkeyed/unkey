package preflight

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/preflight"
)

// Cmd is the preflight command that runs the Kubernetes mutating admission webhook
// for secrets and credentials injection into pods.
var Cmd = &cli.Command{
	Name:  "preflight",
	Usage: "Run the pod mutation webhook for secrets and credentials injection",
	Flags: []cli.Flag{
		cli.Int("http-port", "HTTP port for the webhook server. Default: 8443",
			cli.Default(8443), cli.EnvVar("UNKEY_HTTP_PORT")),
		cli.String("tls-cert-file", "Path to TLS certificate file",
			cli.Required(), cli.EnvVar("UNKEY_TLS_CERT_FILE")),
		cli.String("tls-key-file", "Path to TLS private key file",
			cli.Required(), cli.EnvVar("UNKEY_TLS_KEY_FILE")),
		cli.String("inject-image", "Container image for inject binary",
			cli.Default("inject:latest"), cli.EnvVar("UNKEY_INJECT_IMAGE")),
		cli.String("inject-image-pull-policy", "Image pull policy (Always, IfNotPresent, Never)",
			cli.Default("IfNotPresent"), cli.EnvVar("UNKEY_INJECT_IMAGE_PULL_POLICY")),
		cli.String("krane-endpoint", "Endpoint for Krane secrets service",
			cli.Default("http://krane.unkey.svc.cluster.local:8070"), cli.EnvVar("UNKEY_KRANE_ENDPOINT")),
		cli.String("depot-token", "Depot API token for fetching on-demand pull tokens (optional)",
			cli.EnvVar("UNKEY_DEPOT_TOKEN")),
		cli.StringSlice("insecure-registries", "Comma-separated list of insecure (HTTP) registries",
			cli.EnvVar("UNKEY_INSECURE_REGISTRIES")),
		cli.StringSlice("registry-aliases", "Comma-separated list of registry aliases (from=to)",
			cli.EnvVar("UNKEY_REGISTRY_ALIASES")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	config := preflight.Config{
		HttpPort:              cmd.Int("http-port"),
		TLSCertFile:           cmd.RequireString("tls-cert-file"),
		TLSKeyFile:            cmd.RequireString("tls-key-file"),
		InjectImage:           cmd.String("inject-image"),
		InjectImagePullPolicy: cmd.String("inject-image-pull-policy"),
		KraneEndpoint:         cmd.String("krane-endpoint"),
		DepotToken:            cmd.String("depot-token"),
		InsecureRegistries:    cmd.StringSlice("insecure-registries"),
		RegistryAliases:       cmd.StringSlice("registry-aliases"),
	}

	if err := config.Validate(); err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	return preflight.Run(ctx, config)
}
