package api

import (
	"github.com/unkeyed/unkey/cmd/api/analytics"
	"github.com/unkeyed/unkey/cmd/api/apis"
	"github.com/unkeyed/unkey/cmd/api/apps"
	"github.com/unkeyed/unkey/cmd/api/deployments"
	"github.com/unkeyed/unkey/cmd/api/domains"
	"github.com/unkeyed/unkey/cmd/api/environments"
	"github.com/unkeyed/unkey/cmd/api/gateway"
	"github.com/unkeyed/unkey/cmd/api/github"
	"github.com/unkeyed/unkey/cmd/api/identities"
	"github.com/unkeyed/unkey/cmd/api/keys"
	"github.com/unkeyed/unkey/cmd/api/permissions"
	"github.com/unkeyed/unkey/cmd/api/portal"
	"github.com/unkeyed/unkey/cmd/api/projects"
	"github.com/unkeyed/unkey/cmd/api/ratelimit"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd returns the top-level api command that groups all API management subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "api",
		Usage:       "Interact with the Unkey API",
		Description: "Manage Unkey resources through the public API." + util.Disclaimer,
		Commands: []*cli.Command{
			analytics.Cmd(),
			apis.Cmd(),
			apps.Cmd(),
			deployments.Cmd(),
			domains.Cmd(),
			environments.Cmd(),
			gateway.Cmd(),
			github.Cmd(),
			identities.Cmd(),
			keys.Cmd(),
			permissions.Cmd(),
			portal.Cmd(),
			projects.Cmd(),
			ratelimit.Cmd(),
		},
	}
}
