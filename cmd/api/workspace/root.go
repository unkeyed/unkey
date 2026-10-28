package workspace

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "workspace",
		Usage:       "Manage workspace",
		Description: "Manage workspace through the Unkey API." + util.Disclaimer,
		Commands: []*cli.Command{
			getFlagsCmd(),
			getLimitsCmd(),
			getUsageCmd(),
			updateFlagCmd(),
		},
	}
}
