package sentinel

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/sentinel"
)

// Cmd is the sentinel command that runs the deployment proxy server for routing
// requests to deployment instances within a specific environment.
var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "sentinel",
	Usage:       "Run the Unkey Sentinel server (deployment proxy)",
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[sentinel.Config](cmd.String("config"))
	if err != nil {
		return cli.Exit("Failed to load config: "+err.Error(), 1)
	}

	return sentinel.Run(ctx, cfg)
}
