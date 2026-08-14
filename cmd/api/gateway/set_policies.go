package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func setPoliciesCmd() *cli.Command {
	return &cli.Command{
		Name: "set-policies", Usage: "Replace an environment's gateway policies in a single atomic request",
		Description: `Replace an environment's gateway policies in a single atomic request. Policies run at the edge before requests reach your app: verify API keys, rate limit, block requests outright, or validate them against your OpenAPI spec.

Policies are an ordered list: the gateway evaluates them top to bottom and the first rejection short-circuits the request. Every call is a full, atomic replace; an empty list removes all policies.

Required Permissions
- environment.*.set_policies (for any environment)
- environment.<environment_id>.set_policies (for a specific environment)

For full documentation, see https://www.unkey.com/docs/api-reference/gateway/set-policies` + util.Disclaimer,
		Examples: []string{`unkey api gateway set-policies --project=payments --app=payments-api --environment=production --policies='[{"name":"Require API key","enabled":true,"keyauth":{"keyspaces":["ks_1234abcd"]}}]'`, `unkey api gateway set-policies --project=payments --app=payments-api --environment=production --policies='[]'`},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("policies", "JSON array containing the complete ordered policy list.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Gateway.SetPolicies, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2GatewaySetPoliciesResponseBody)
			}
			send := func(req components.V2GatewaySetPoliciesRequestBody) error {
				res, err := client.Gateway.SetPolicies(ctx, req)
				if err != nil {
					return fmt.Errorf("%s", util.FormatError(err))
				}
				return util.Output(cmd, res.V2GatewaySetPoliciesResponseBody)
			}
			var policies []components.Policy
			rawPolicies := cmd.String("policies")
			if strings.TrimSpace(rawPolicies) == "null" {
				return fmt.Errorf("--policies must be a JSON array")
			}
			if err := json.Unmarshal([]byte(rawPolicies), &policies); err != nil {
				return fmt.Errorf("invalid JSON for --policies: %w", err)
			}
			req := components.V2GatewaySetPoliciesRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), Policies: policies}
			return send(req)
		},
	}
}
