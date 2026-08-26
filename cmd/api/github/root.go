package github

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all github.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "github",
		Usage:       "Manage GitHub App installations",
		Description: "Manage the GitHub App installation for your workspace." + util.Disclaimer,
		Commands: []*cli.Command{
			installAppCmd(),
		},
	}
}
