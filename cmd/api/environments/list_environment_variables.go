package environments

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listEnvironmentVariablesCmd() *cli.Command {
	return &cli.Command{
		Name: "list-environment-variables", Usage: "List an environment's variables", Description: "List environment variables in pages of up to 100 entries by default. Use the cursor returned by a response to fetch the next page.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/environments/list-environment-variables" + util.Disclaimer,
		Examples: []string{"unkey api environments list-environment-variables --project=payments --app=payments-api --environment=production", "unkey api environments list-environment-variables --project=payments --app=payments-api --environment=production --limit=25"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.Int64("limit", "Maximum variables to return per page (default 100).", cli.MutuallyExclusive("body")), cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Environments.ListEnvironmentVariables, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2EnvironmentsListEnvironmentVariablesResponseBody)
			}
			req := components.V2EnvironmentsListEnvironmentVariablesRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), Limit: nil, Cursor: nil}
			if v := cmd.Int64("limit"); v != 0 {
				req.Limit = &v
			}
			if v := cmd.String("cursor"); v != "" {
				req.Cursor = &v
			}
			res, err := client.Environments.ListEnvironmentVariables(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2EnvironmentsListEnvironmentVariablesResponseBody)
		},
	}
}
