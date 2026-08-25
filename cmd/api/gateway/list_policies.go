package gateway

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listPoliciesCmd() *cli.Command {
	return &cli.Command{
		Name: "list-policies", Usage: "Retrieve an environment's gateway policies in evaluation order",
		Description: `Retrieve an environment's gateway policies in evaluation order: the gateway evaluates them top to bottom and the first rejection short-circuits the request.

The full policy list is returned in a single response.

Required Permissions

Your root key must have one of the following permissions:
- environment.*.read_policies (for any environment)
- environment.<environment_id>.read_policies (for a specific environment)

For full documentation, see https://www.unkey.com/docs/api-reference/gateway/list-policies` + util.Disclaimer,
		Examples: []string{"unkey api gateway list-policies --project=payments --app=payments-api --environment=production"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Gateway.ListPolicies, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2GatewayListPoliciesResponseBody)
			}
			req := components.V2GatewayListPoliciesRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment")}
			res, err := client.Gateway.ListPolicies(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2GatewayListPoliciesResponseBody)
		},
	}
}
