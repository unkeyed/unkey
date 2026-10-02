package deployments

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createDeploymentCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-deployment",
		Usage: "Create a deployment for an app in a project.",
		Description: `Create a deployment for an app in a project.

Omit the source to use the app's configured default. A Git app builds its default branch. An OCI app deploys its default image.

Optionally provide one source override:
- oci: deploy a prebuilt OCI image without a build. Mutable tags are resolved to immutable digests before rollout.
- git: build and deploy from the app's connected GitHub repository, a branch, a specific commit, or a fork commit. Requires the app to have a repository connected.
- deployment: re-run an existing deployment by its id. Git deployments rebuild from the recorded commit; OCI deployments reuse the recorded resolved image.

Returns immediately with a deploymentId. The build and rollout run asynchronously. Poll deployments.getDeployment to watch status until it is ready.

Authentication: requires a root key with permission to create deployments.

For full documentation, see https://www.unkey.com/docs/api-reference/deployments/create-deployment` + util.Disclaimer,
		Examples: []string{
			"unkey api deployments create-deployment --project=payments --app=payments-api --environment=production",
			`unkey api deployments create-deployment --project=payments --app=payments-api --environment=production --git='{"branch":"main"}'`,
			`unkey api deployments create-deployment --project=payments --app=payments-api --environment=production --oci='{"image":"ghcr.io/acme/payments:v1.2.3"}'`,
			`unkey api deployments create-deployment --project=payments --app=payments-api --environment=production --deployment='{"deploymentId":"d_abc123xyz"}'`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("git", "Build from the app's connected GitHub repository using a JSON object.", cli.MutuallyExclusive("body", "oci", "deployment")),
			cli.String("oci", "Deploy a prebuilt OCI image without a build using a JSON object.", cli.MutuallyExclusive("body", "git", "deployment")),
			cli.String("deployment", "Re-run an existing deployment using a JSON object.", cli.MutuallyExclusive("body", "git", "oci")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Deployments.CreateDeploymentV3, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V3DeploymentsCreateDeploymentResponseBody)
			}

			req := components.V3DeploymentsCreateDeploymentRequestBody{
				Project:     cmd.String("project"),
				App:         cmd.String("app"),
				Environment: cmd.String("environment"),
				Git:         nil,
				Oci:         nil,
				Deployment:  nil,
			}
			if cmd.String("git") != "" {
				var source *components.DeploymentSourceGit
				if err := cmd.JSON("git", &source); err != nil {
					return err
				}
				if source == nil {
					return fmt.Errorf("--git must be a JSON object, not null")
				}
				req.Git = source
			}
			if cmd.String("oci") != "" {
				var source *components.DeploymentSourceOCI
				if err := cmd.JSON("oci", &source); err != nil {
					return err
				}
				if source == nil {
					return fmt.Errorf("--oci must be a JSON object, not null")
				}
				req.Oci = source
			}
			if cmd.String("deployment") != "" {
				var source *components.DeploymentSourceDeployment
				if err := cmd.JSON("deployment", &source); err != nil {
					return err
				}
				if source == nil {
					return fmt.Errorf("--deployment must be a JSON object, not null")
				}
				req.Deployment = source
			}
			sources := 0
			for _, present := range []bool{req.Git != nil, req.Oci != nil, req.Deployment != nil} {
				if present {
					sources++
				}
			}
			if sources > 1 {
				return fmt.Errorf("only one of --git, --oci, or --deployment may be provided")
			}
			res, err := client.Deployments.CreateDeploymentV3(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V3DeploymentsCreateDeploymentResponseBody)
		},
	}
}
