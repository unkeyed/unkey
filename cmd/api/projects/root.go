package projects

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all projects.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{Name: "projects", Usage: "Manage projects", Description: "Create, read, update, and delete workspace projects." + util.Disclaimer, Commands: []*cli.Command{
		createProjectCmd(), deleteProjectCmd(), getProjectCmd(), listProjectsCmd(), updateProjectCmd(),
	}}
}
