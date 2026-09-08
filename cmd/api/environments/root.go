package environments

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd returns the environments group command.
func Cmd() *cli.Command {
	return &cli.Command{Name: "environments", Usage: "Manage environments", Description: "Manage Unkey environments." + util.Disclaimer, Commands: []*cli.Command{
		setEnvironmentVariablesCmd(),
		updateSettingsCmd(),
		getEnvironmentCmd(),
		listEnvironmentVariablesCmd(),
		listEnvironmentsCmd(),
		removeEnvironmentVariablesCmd(),
	}}
}
