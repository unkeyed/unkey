package vault

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
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
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[vault.Config](cmd.String("config"))
	if err != nil {
		return fmt.Errorf("unable to load config: %w")
	}

	return vault.Run(ctx, cfg)
}
