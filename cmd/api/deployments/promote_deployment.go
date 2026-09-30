package deployments

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func promoteDeploymentCmd() *cli.Command {
	return &cli.Command{
		Name: "promote-deployment", Usage: "Promote a deployment to production", Description: "Promote an existing deployment to its app's production environment. The promotion runs asynchronously.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/deployments/promote-deployment" + util.Disclaimer,
		Examples: []string{"unkey api deployments promote-deployment --deployment-id=dep_1234abcd"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("deployment-id", "Unique deployment ID to promote.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Deployments.PromoteDeployment, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsPromoteDeploymentResponseBody)
			}
			req := components.V2DeploymentsPromoteDeploymentRequestBody{DeploymentID: cmd.String("deployment-id")}
			res, err := client.Deployments.PromoteDeployment(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2DeploymentsPromoteDeploymentResponseBody)
		},
	}
}
