package identities

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all identities.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "identities",
		Usage:       "Manage identities",
		Description: "Create, read, update, and delete identities for grouping API keys." + util.Disclaimer,
		Commands: []*cli.Command{
			createIdentityCmd(),
			deleteIdentityCmd(),
			getIdentityCmd(),
			listIdentitiesCmd(),
			updateIdentityCmd(),
		},
	}
}
