package deployments

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func rollbackDeploymentCmd() *cli.Command {
	return &cli.Command{
		Name: "rollback-deployment", Usage: "Roll back to an earlier deployment", Description: "Create an asynchronous rollback using the selected deployment as the source.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/deployments/rollback-deployment" + util.Disclaimer,
		Examples: []string{"unkey api deployments rollback-deployment --deployment-id=dep_1234abcd"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("deployment-id", "Unique deployment ID to restore.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Deployments.RollbackDeployment, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsRollbackDeploymentResponseBody)
			}
			req := components.V2DeploymentsRollbackDeploymentRequestBody{DeploymentID: cmd.String("deployment-id")}
			res, err := client.Deployments.RollbackDeployment(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2DeploymentsRollbackDeploymentResponseBody)
		},
	}
}
