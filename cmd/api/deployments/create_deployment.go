package deployments

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createDeploymentCmd() *cli.Command {
	return &cli.Command{
		Name: "create-deployment", Usage: "Create a deployment from Git, an image, or an existing deployment",
		Description: "Create a deployment asynchronously from exactly one source: a Git branch, a container image, or an existing deployment. The response includes a deployment ID that you can pass to get-deployment to monitor progress.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/deployments/create-deployment" + util.Disclaimer,
		Examples:    []string{"unkey api deployments create-deployment --project=payments --app=payments-api --environment=production --git='{" + `"branch":"main"` + "}'"},
		Flags:       []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("git", "Git source as JSON.", cli.MutuallyExclusive("body")), cli.String("image", "Image source as JSON.", cli.MutuallyExclusive("body")), cli.String("deployment", "Existing deployment source as JSON.", cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Deployments.CreateDeployment, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsCreateDeploymentResponseBody)
			}
			send := func(req components.V2DeploymentsCreateDeploymentRequestBodyUnion) error {
				res, err := client.Deployments.CreateDeployment(ctx, req)
				if err != nil {
					return fmt.Errorf("%s", util.FormatError(err))
				}
				return util.Output(cmd, res.V2DeploymentsCreateDeploymentResponseBody)
			}
			project, app, environment := cmd.String("project"), cmd.String("app"), cmd.String("environment")
			var req components.V2DeploymentsCreateDeploymentRequestBodyUnion
			sources := 0
			if raw := cmd.String("git"); raw != "" {
				var source *components.DeploymentSourceGit
				if err := json.Unmarshal([]byte(raw), &source); err != nil {
					return fmt.Errorf("invalid JSON for --git: %w", err)
				}
				if source == nil {
					return fmt.Errorf("--git must be a JSON object, not null")
				}
				req = components.CreateV2DeploymentsCreateDeploymentRequestBodyUnionV2DeploymentsCreateDeploymentRequestBody2(components.V2DeploymentsCreateDeploymentRequestBody2{Project: project, App: app, Environment: environment, Git: *source, Image: nil, Deployment: nil})
				sources++
			}
			if raw := cmd.String("image"); raw != "" {
				var source *components.DeploymentSourceImage
				if err := json.Unmarshal([]byte(raw), &source); err != nil {
					return fmt.Errorf("invalid JSON for --image: %w", err)
				}
				if source == nil {
					return fmt.Errorf("--image must be a JSON object, not null")
				}
				req = components.CreateV2DeploymentsCreateDeploymentRequestBodyUnionV2DeploymentsCreateDeploymentRequestBody1(components.V2DeploymentsCreateDeploymentRequestBody1{Project: project, App: app, Environment: environment, Git: nil, Image: *source, Deployment: nil})
				sources++
			}
			if raw := cmd.String("deployment"); raw != "" {
				var source *components.DeploymentSourceDeployment
				if err := json.Unmarshal([]byte(raw), &source); err != nil {
					return fmt.Errorf("invalid JSON for --deployment: %w", err)
				}
				if source == nil {
					return fmt.Errorf("--deployment must be a JSON object, not null")
				}
				req = components.CreateV2DeploymentsCreateDeploymentRequestBodyUnionV2DeploymentsCreateDeploymentRequestBody3(components.V2DeploymentsCreateDeploymentRequestBody3{Project: project, App: app, Environment: environment, Git: nil, Image: nil, Deployment: *source})
				sources++
			}
			if sources != 1 {
				return fmt.Errorf("exactly one of --git, --image, or --deployment is required")
			}
			return send(req)
		},
	}
}
