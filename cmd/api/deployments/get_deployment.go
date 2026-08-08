package deployments

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getDeploymentCmd() *cli.Command {
	return &cli.Command{
		Name: "get-deployment", Usage: "Retrieve a deployment by ID", Description: "Retrieve deployment details, lifecycle status, build steps, and runtime information by deployment ID.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/deployments/get-deployment" + util.Disclaimer,
		Examples: []string{"unkey api deployments get-deployment --deployment-id=dep_1234abcd"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("deployment-id", "Unique deployment ID.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Deployments.GetDeployment, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsGetDeploymentResponseBody)
			}
			req := components.V2DeploymentsGetDeploymentRequestBody{DeploymentID: cmd.String("deployment-id")}
			res, err := client.Deployments.GetDeployment(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2DeploymentsGetDeploymentResponseBody)
		},
	}
}
