package apps

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all apps.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{Name: "apps", Usage: "Manage apps", Description: "Create, read, update, and delete apps within projects." + util.Disclaimer, Commands: []*cli.Command{
		createAppCmd(), deleteAppCmd(), getAppCmd(), listAppsCmd(), updateAppCmd(),
	}}
}
