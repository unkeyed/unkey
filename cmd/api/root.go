package api

import (
	"github.com/unkeyed/unkey/cmd/api/analytics"
	"github.com/unkeyed/unkey/cmd/api/apis"
	"github.com/unkeyed/unkey/cmd/api/identities"
	"github.com/unkeyed/unkey/cmd/api/keys"
	"github.com/unkeyed/unkey/cmd/api/permissions"
	"github.com/unkeyed/unkey/cmd/api/ratelimit"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd returns the top-level api command that groups all API management subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "api",
		Usage:       "Interact with the Unkey API",
		Description: "Manage APIs, keys, identities, permissions, rate limits, and analytics." + util.Disclaimer,
		Commands: []*cli.Command{
			analytics.Cmd(),
			apis.Cmd(),
			identities.Cmd(),
			keys.Cmd(),
			permissions.Cmd(),
			ratelimit.Cmd(),
		},
	}
}
