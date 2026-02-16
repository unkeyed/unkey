package preflight

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/preflight"
)

// Cmd is the preflight command that runs the Kubernetes mutating admission webhook
// for secrets and credentials injection into pods.
var Cmd = &cli.Command{
	Name:  "preflight",
	Usage: "Run the pod mutation webhook for secrets and credentials injection",
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[preflight.Config](cmd.String("config"))
	if err != nil {
		return cli.Exit("Failed to load config: "+err.Error(), 1)
	}

	return preflight.Run(ctx, cfg)
}
