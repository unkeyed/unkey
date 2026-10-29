package version

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd reports the version of the installed CLI.
var Cmd = &cli.Command{
	Name:        "version",
	Usage:       "Show the CLI version",
	Description: "Print the installed CLI version. Use 'unkey deployments' to manage deployments.",
	Action: func(context.Context, *cli.Command) error {
		_, err := fmt.Fprintln(os.Stdout, buildinfo.Version)
		return err
	},
}
