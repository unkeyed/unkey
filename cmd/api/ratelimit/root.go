package ratelimit

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all ratelimit.* subcommands.
var Cmd = &cli.Command{
	Name:        "ratelimit",
	Usage:       "Manage rate limiting",
	Description: "Apply rate limits and manage overrides." + util.Disclaimer,
	Commands: []*cli.Command{
		limitCmd,
		multiLimitCmd,
		setOverrideCmd,
		getOverrideCmd,
		deleteOverrideCmd,
		listOverridesCmd,
	},
}
