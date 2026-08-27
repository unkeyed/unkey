package apps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createAppCmd() *cli.Command {
	return &cli.Command{Name: "create-app", Usage: "Create an app within a project.", Description: `Create an app within a project. The app is created with default production and preview environments.

The slug you provide is the stable, caller-defined handle used to reference this app. It must be unique within the project.

Important: The slug cannot collide with an existing app in the same project. A duplicate slug returns a 409 conflict.

Required Permissions
- project.*.create_app (to create apps in any project)
- project.<project_id>.create_app (to create apps in a specific project)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apps/create-app` + util.Disclaimer, Examples: []string{"unkey api apps create-app --project=payments --name='Payments API' --slug=payments-api", `unkey api apps create-app --project=payments --name='Payments API' --slug=payments-api --git='{"repository":"unkeyed/api","defaultBranch":"main"}'`}, Flags: []cli.Flag{
		cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
		util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(),
		cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("name", "Human-readable name for this app.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("slug", "Stable app slug.", cli.Required(), cli.MutuallyExclusive("body")),
		cli.String("git", "GitHub repository connection as a JSON object.", cli.MutuallyExclusive("body")),
	}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Apps.CreateApp, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2AppsCreateAppResponseBody)
		}
		req := components.V2AppsCreateAppRequestBody{Project: cmd.String("project"), Name: cmd.String("name"), Slug: cmd.String("slug"), Git: nil}
		if v := cmd.String("git"); v != "" {
			if err := json.Unmarshal([]byte(v), &req.Git); err != nil {
				return fmt.Errorf("invalid JSON for --git: %w", err)
			}
		}
		res, err := client.Apps.CreateApp(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2AppsCreateAppResponseBody)
	}}
}
