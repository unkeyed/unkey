package deployments

import (
	"context"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func cancelDeploymentCmd() *cli.Command {
	return &cli.Command{
		Name:  "cancel-deployment",
		Usage: "Cancel a deployment that has not finished yet: queued, waiting for approval, building, or deploying",
		Description: `Cancel a deployment that has not finished yet: queued, waiting for
approval, building, or deploying. Cancelling aborts its run and marks it
cancelled.

Only in-progress deployments can be cancelled. To take a ready preview
deployment offline, use stopDeployment instead.

Cancellation is asynchronous: this endpoint only enqueues it and returns
immediately. Poll getDeployment until the status reaches cancelled.

Required permissions:
- unkey:v1:<workspace_id>:projects/<project_id>/apps/<app_id>/environments/<environment_id>/deployments/<deployment_id>#write

For full documentation, see https://www.unkey.com/docs/compute/cli/deployments/cancel-deployment` + util.Disclaimer,
		Examples: []string{
			"unkey api deployments cancel-deployment --deployment-id=d_1234abcd",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("deployment-id", "The deployment ID or slug to cancel.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Deployments.CancelDeployment, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsCancelDeploymentResponseBody)
			}

			req := components.V2DeploymentsCancelDeploymentRequestBody{
				DeploymentID: cmd.String("deployment-id"),
			}
			res, err := client.Deployments.CancelDeployment(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2DeploymentsCancelDeploymentResponseBody)
		},
	}
}
