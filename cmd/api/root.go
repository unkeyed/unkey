package api

import (
	"context"

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
	"github.com/unkeyed/unkey/cmd/api/logs"
	"github.com/unkeyed/unkey/cmd/api/permissions"
	"github.com/unkeyed/unkey/cmd/api/portal"
	"github.com/unkeyed/unkey/cmd/api/projects"
	"github.com/unkeyed/unkey/cmd/api/ratelimit"
	"github.com/unkeyed/unkey/cmd/api/rootkeys"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/cmd/api/workspace"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd returns the top-level api command that groups all API management subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "api",
		Usage:       "Legacy API command paths",
		Description: "Manage Unkey resources through the public API." + util.Disclaimer,
		Commands:    groups(),
		OnError:     HandleError,
	}
}

func HandleError(_ context.Context, cmd *cli.Command, args []string, err error) error {
	return util.CommandError(cmd, "", err, args...)
}

func groups() []*cli.Command {
	commands := []*cli.Command{
		analytics.Cmd(), apis.Cmd(), apps.Cmd(), deployments.Cmd(), domains.Cmd(),
		environments.Cmd(), gateway.Cmd(), github.Cmd(), identities.Cmd(), keys.Cmd(),
		logs.Cmd(), permissions.Cmd(), portal.Cmd(), projects.Cmd(), ratelimit.Cmd(),
		rootkeys.Cmd(), workspace.Cmd(),
	}
	for _, group := range commands {
		for _, command := range group.Commands {
			path := resourcePaths[group.Name+" "+command.Name]
			command.OnError = func(_ context.Context, cmd *cli.Command, args []string, err error) error {
				return util.CommandError(cmd, path, err, args...)
			}
			command.Flags = append(command.Flags,
				cli.Bool("json", "Print the full API response as JSON"),
				cli.Bool("debug", "Print the request ID to stderr"),
				cli.Enum("error-format", "Error format on stderr", []string{"human", "json"}, cli.Default("human")),
			)
		}
	}
	return commands
}
