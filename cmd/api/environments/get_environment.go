package environments

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getEnvironmentCmd() *cli.Command {
	return &cli.Command{
		Name: "get-environment", Usage: "Retrieve an environment by ID or slug", Description: "Retrieve an environment and its build and runtime settings within an app. Project, app, and environment selectors accept IDs or slugs.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/environments/get-environment" + util.Disclaimer,
		Examples: []string{"unkey api environments get-environment --project=payments --app=payments-api --environment=production"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Environments.GetEnvironment, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2EnvironmentsGetEnvironmentResponseBody)
			}
			req := components.V2EnvironmentsGetEnvironmentRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment")}
			res, err := client.Environments.GetEnvironment(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2EnvironmentsGetEnvironmentResponseBody)
		},
	}
}
