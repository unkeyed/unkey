package projects

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func updateProjectCmd() *cli.Command {
	return &cli.Command{Name: "update-project", Usage: "Update an existing project in your workspace, identified by its id.", Description: `Update an existing project in your workspace, identified by its id.

The project name, slug, and delete protection setting can be changed. Omitted fields are left unchanged. Changing the slug affects the deployment domains generated for this project.

Required Permissions
- project.*.update_project (to update any project)
- project.<project_id>.update_project (to update a specific project)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/projects/update-project` + util.Disclaimer, Examples: []string{"unkey api projects update-project --project=proj_1234abcd --name='Payments API'", "unkey api projects update-project --project=payments --delete-protection=true"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("slug", "New project slug.", cli.MutuallyExclusive("body")), cli.String("name", "New human-readable name for the project.", cli.MutuallyExclusive("body")), cli.Bool("delete-protection", "Enable or disable delete protection for the project.", cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Projects.UpdateProject, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2ProjectsUpdateProjectResponseBody)
		}
		req := components.V2ProjectsUpdateProjectRequestBody{Project: cmd.String("project"), Slug: nil, Name: nil, DeleteProtection: nil}
		if v := cmd.String("slug"); v != "" {
			req.Slug = &v
		}
		if v := cmd.String("name"); v != "" {
			req.Name = &v
		}
		if cmd.FlagIsSet("delete-protection") {
			req.DeleteProtection = ptr.P(cmd.Bool("delete-protection"))
		}
		res, err := client.Projects.UpdateProject(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2ProjectsUpdateProjectResponseBody)
	}}
}
