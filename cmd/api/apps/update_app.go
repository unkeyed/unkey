package apps

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func updateAppCmd() *cli.Command {
	return &cli.Command{Name: "update-app", Usage: "Update an existing app, identified by its id.", Description: `Update an existing app, identified by its id.

The app name, slug, default branch, and delete protection setting can be changed. Omitted fields are left unchanged. Changing the slug affects the deployment domains generated for this app.

Important: The slug cannot collide with an existing app in the same project. A duplicate slug returns a 409 conflict.

Required Permissions
- app.*.update_app (to update any app)
- app.<app_id>.update_app (to update a specific app)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apps/update-app` + util.Disclaimer, Examples: []string{"unkey api apps update-app --project=payments --app=app_1234abcd --name='Payments API'", "unkey api apps update-app --project=payments --app=payments-api --delete-protection=true"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("name", "New human-readable name for the app.", cli.MutuallyExclusive("body")), cli.String("slug", "New app slug.", cli.MutuallyExclusive("body")), cli.String("default-branch", "New default git branch deployments track.", cli.MutuallyExclusive("body")), cli.Bool("delete-protection", "Enable or disable delete protection for the app.", cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Apps.UpdateApp, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2AppsUpdateAppResponseBody)
		}
		req := components.V2AppsUpdateAppRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Name: nil, Slug: nil, DefaultBranch: nil, DeleteProtection: nil}
		if v := cmd.String("name"); v != "" {
			req.Name = &v
		}
		if v := cmd.String("slug"); v != "" {
			req.Slug = &v
		}
		if v := cmd.String("default-branch"); v != "" {
			req.DefaultBranch = &v
		}
		if cmd.FlagIsSet("delete-protection") {
			req.DeleteProtection = ptr.P(cmd.Bool("delete-protection"))
		}
		res, err := client.Apps.UpdateApp(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2AppsUpdateAppResponseBody)
	}}
}
