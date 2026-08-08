package apps

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getAppCmd() *cli.Command {
	return &cli.Command{Name: "get-app", Usage: "Retrieve a single app by its id or slug within a project.", Description: `Retrieve a single app by its id or slug within a project.

Use this to fetch app details after creation or to verify an app exists before performing operations.

Required Permissions
- app.*.read_app (to read any app)
- app.<app_id>.read_app (to read a specific app)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apps/get-app` + util.Disclaimer, Examples: []string{"unkey api apps get-app --project=payments --app=app_1234abcd"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Apps.GetApp, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2AppsGetAppResponseBody)
		}
		req := components.V2AppsGetAppRequestBody{Project: cmd.String("project"), App: cmd.String("app")}
		res, err := client.Apps.GetApp(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2AppsGetAppResponseBody)
	}}
}
