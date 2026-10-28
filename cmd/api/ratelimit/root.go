package ratelimit

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all ratelimit.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "ratelimit",
		Usage:       "Manage rate limiting",
		Description: "Apply rate limits and manage namespaces and overrides." + util.Disclaimer,
		Commands: []*cli.Command{
			createNamespaceCmd(),
			deleteNamespaceCmd(),
			listNamespacesCmd(),
			updateNamespaceCmd(),
			limitCmd(),
			multiLimitCmd(),
			setOverrideCmd(),
			getOverrideCmd(),
			deleteOverrideCmd(),
			listOverridesCmd(),
		},
	}
}
