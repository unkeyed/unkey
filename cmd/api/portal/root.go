package portal

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func Cmd() *cli.Command {
	return &cli.Command{Name: "portal", Usage: "Manage Customer Portal sessions", Description: "Create Customer Portal sessions using bearer authentication." + util.Disclaimer, Commands: []*cli.Command{createSessionCmd()}}
}
