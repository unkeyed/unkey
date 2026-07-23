package projects

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createProjectCmd() *cli.Command {
	return &cli.Command{Name: "create-project", Usage: "Create a project to group deployments and applications under a workspace-scoped slug.", Description: `Create a project to group deployments and applications under a workspace-scoped slug.

The slug you provide is the stable, caller-defined handle used to reference this project in subsequent operations (get, update, delete). It must be unique within your workspace.

Important: The slug cannot collide with an existing project in your workspace. A duplicate slug returns a 409 conflict.

Required Permissions
- project.*.create_project (to create projects in your workspace)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/projects/create-project` + util.Disclaimer, Examples: []string{"unkey api projects create-project --name='Payments Service' --slug=payments-service"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("name", "Human-readable name for this project.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("slug", "Stable project slug.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Projects.CreateProject, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2ProjectsCreateProjectResponseBody)
		}
		req := components.V2ProjectsCreateProjectRequestBody{Name: cmd.String("name"), Slug: cmd.String("slug")}
		res, err := client.Projects.CreateProject(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2ProjectsCreateProjectResponseBody)
	}}
}
