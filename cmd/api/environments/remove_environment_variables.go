package environments

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func removeEnvironmentVariablesCmd() *cli.Command {
	return &cli.Command{
		Name:        "remove-environment-variables",
		Usage:       "Remove variables from an environment",
		Description: "Remove the named environment variables from an environment in one request. Variables not named by the command remain unchanged.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/environments/remove-environment-variables" + util.Disclaimer,
		Examples: []string{
			"unkey api environments remove-environment-variables --project=payments --app=payments-api --environment=production --variables=OLD_TOKEN,LEGACY_URL",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.StringSlice("variables", "Comma-separated variable names to remove.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2EnvironmentsRemoveEnvironmentVariablesRequestBody{
				Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), Variables: cmd.StringSlice("variables"),
			})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Remove variables %q from environment %q, app %q, project %q?", req.Variables, req.Environment, req.App, req.Project)); err != nil {
				return err
			}
			res, err := client.Environments.RemoveEnvironmentVariables(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2EnvironmentsRemoveEnvironmentVariablesResponseBody)
		},
	}
}
