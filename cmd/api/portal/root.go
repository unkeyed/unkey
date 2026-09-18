package portal

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func Cmd() *cli.Command {
	return &cli.Command{Name: "portal", Usage: "Manage Customer Portals and sessions", Description: "Create and manage Customer Portals and sessions using bearer authentication." + util.Disclaimer, Commands: []*cli.Command{createPortalCmd(), createSessionCmd(), deletePortalCmd(), getPortalCmd(), updatePortalCmd()}}
}
