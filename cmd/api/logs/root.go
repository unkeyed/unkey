package logs

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "logs",
		Usage:       "Manage logs",
		Description: "Manage logs through the Unkey API." + util.Disclaimer,
		Commands: []*cli.Command{
			listAPIRequestsCmd(),
			getAPIRequestMetricsCmd(),
		},
	}
}
