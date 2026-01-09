package main

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/secrets/provider"
)

var cmd = &cli.Command{
	Name:        "inject",
	Usage:       "Fetch secrets and exec the given command",
	AcceptsArgs: true,
	Flags: []cli.Flag{
		cli.String("provider", "Secrets provider type",
			cli.Default(string(provider.KraneVault)),
			cli.EnvVar("UNKEY_PROVIDER")),
		cli.String("endpoint", "Provider API endpoint",
			cli.EnvVar("UNKEY_PROVIDER_ENDPOINT")),
		cli.String("deployment-id", "Deployment ID",
			cli.EnvVar("UNKEY_DEPLOYMENT_ID")),
		cli.String("environment-id", "Environment ID for decryption",
			cli.EnvVar("UNKEY_ENVIRONMENT_ID")),
		cli.String("secrets-blob", "Base64-encoded encrypted secrets blob",
			cli.EnvVar("UNKEY_ENCRYPTED_ENV")),
		cli.String("token", "Authentication token",
			cli.EnvVar("UNKEY_TOKEN")),
		cli.String("token-path", "Path to token file",
			cli.EnvVar("UNKEY_TOKEN_PATH")),
		cli.Bool("debug", "Enable debug logging",
			cli.EnvVar("UNKEY_DEBUG")),
	},
	Action: action,
}

func action(ctx context.Context, c *cli.Command) error {
	cfg := config{
		Provider:      provider.Type(c.String("provider")),
		Endpoint:      c.String("endpoint"),
		DeploymentID:  c.String("deployment-id"),
		EnvironmentID: c.String("environment-id"),
		Encrypted:     c.String("secrets-blob"),
		Token:         c.String("token"),
		TokenPath:     c.String("token-path"),
		Debug:         c.Bool("debug"),
		Args:          c.Args(),
	}

	if err := cfg.validate(); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return run(ctx, cfg)
}
