package deploy

import (
	"github.com/unkeyed/unkey/go/cmd/version"
	"github.com/urfave/cli/v3"
)

// Cmd is an alias for "version create" to provide a more intuitive top-level command
var Cmd = &cli.Command{
	Name:  "deploy",
	Usage: "Deploy a new version of your API (alias for 'version create')",
	Description: `Deploy is a convenience command that creates a new version of your API.
	
This is equivalent to running 'unkey version create'.`,

	// Copy all flags from version create command
	Flags:  version.Cmd.Commands[0].Flags,
	Action: version.Cmd.Commands[0].Action,
}
