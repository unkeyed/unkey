package deployments

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func authorizeDeploymentCmd() *cli.Command {
	return &cli.Command{
		Name:  "authorize-deployment",
		Usage: "Authorize a deployment that is awaiting approval",
		Description: `Authorize a deployment that is awaiting approval. Deployments from forked
pull requests run code from outside your repository, so they wait for a
project member to approve them before they build.

The deployment must have the status awaiting_approval. Authorization is
asynchronous: this endpoint only enqueues it and returns immediately. The
deployment then moves to pending and starts its build. Poll
getDeployment to follow its status. If the build cannot start, the
deployment returns to awaiting_approval.

Required permissions:
- unkey:v1:<workspace_id>:projects/<project_id>/apps/<app_id>/environments/<environment_id>/deployments/<deployment_id>#write

For full documentation, see https://www.unkey.com/docs/compute/cli/deployments/authorize-deployment` + util.Disclaimer,
		Examples: []string{
			"unkey api deployments authorize-deployment --deployment-id=d_1234abcd",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("deployment-id", "The deployment ID or slug to authorize.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Deployments.AuthorizeDeployment, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsAuthorizeDeploymentResponseBody)
			}

			req := components.V2DeploymentsAuthorizeDeploymentRequestBody{
				DeploymentID: cmd.String("deployment-id"),
			}
			res, err := client.Deployments.AuthorizeDeployment(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2DeploymentsAuthorizeDeploymentResponseBody)
		},
	}
}
