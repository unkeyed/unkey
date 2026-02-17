package frontline

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
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
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[frontline.Config](cmd.String("config"))
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	if cfg.FrontlineID == "" {
		cfg.FrontlineID = uid.New("frontline", 4)
	}

	return frontline.Run(ctx, cfg)
}
