package apps

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteAppCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-app",
		Usage: "Delete an existing app, identified by its id.",
		Description: `Delete an existing app, identified by its id.

Deletion is asynchronous and eventually consistent. The app and all of its associated resources (environments, deployments, custom domains) are torn down by a background workflow. A successful response indicates the deletion was enqueued, not that every resource has already been removed.

Apps with delete protection enabled cannot be deleted until protection is disabled.

Required Permissions
- app.*.delete_app (to delete any app)
- app.<app_id>.delete_app (to delete a specific app)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apps/delete-app` + util.Disclaimer,
		Examples: []string{
			"unkey api apps delete-app --project=payments --app=app_1234abcd",
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2AppsDeleteAppRequestBody{Project: cmd.String("project"), App: cmd.String("app")})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Delete app %q in project %q and all its environments, deployments, and domains?", req.App, req.Project)); err != nil {
				return err
			}
			res, err := client.Apps.DeleteApp(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2AppsDeleteAppResponseBody)
		},
	}
}
