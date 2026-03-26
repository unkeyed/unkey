package analytics

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all analytics.* subcommands.
var Cmd = &cli.Command{
	Name:        "analytics",
	Usage:       "Query analytics data",
	Description: "Query key verification analytics with SQL." + util.Disclaimer,
	Commands: []*cli.Command{
		getVerificationsCmd,
	},
}
