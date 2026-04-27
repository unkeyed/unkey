package outpost

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/outpost"
)

var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "outpost",
	Usage:       "Run the Unkey Outpost egress proxy",
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
		cli.String("config-data", "Inline TOML config content (takes precedence over --config)",
			cli.EnvVar("UNKEY_CONFIG_DATA")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	var (
		cfg outpost.Config
		err error
	)

	if data := cmd.String("config-data"); data != "" {
		cfg, err = config.LoadBytes[outpost.Config]([]byte(data))
	} else {
		cfg, err = config.Load[outpost.Config](cmd.String("config"))
	}
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	return outpost.Run(ctx, cfg)
}
