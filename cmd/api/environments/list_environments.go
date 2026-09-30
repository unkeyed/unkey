package environments

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listEnvironmentsCmd() *cli.Command {
	return &cli.Command{
		Name: "list-environments", Usage: "List environments for an app", Description: "List the production and preview environments configured for an app. Project and app selectors accept IDs or slugs.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/environments/list-environments" + util.Disclaimer,
		Examples: []string{"unkey api environments list-environments --project=payments --app=payments-api"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Environments.ListEnvironments, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2EnvironmentsListEnvironmentsResponseBody)
			}
			req := components.V2EnvironmentsListEnvironmentsRequestBody{Project: cmd.String("project"), App: cmd.String("app")}
			res, err := client.Environments.ListEnvironments(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2EnvironmentsListEnvironmentsResponseBody)
		},
	}
}
