package ratelimit

import (
	"context"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func updateNamespaceCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-namespace",
		Usage: "Rename a rate limit namespace, identified by its ID or current name",
		Description: `Rename a rate limit namespace, identified by its ID or current name.

Renaming to the current name returns the namespace without a write.

Important: Overrides stay with the namespace. Requests that pass the namespace by name, such as ratelimit.limit, must use the new name. A ratelimit.limit call with the old name can create a new, empty namespace with that name. A name that is in use or was deleted returns 409. Other API nodes can take up to a minute to see the new name.

Required permissions:
- unkey:v1:<workspace_id>:projects/*/ratelimits/namespaces/*#write (to rename any namespace)
- unkey:v1:<workspace_id>:projects/<project_id>/ratelimits/namespaces/<namespace_id>#write (to rename a specific namespace)

For full documentation, see https://www.unkey.com/docs/api-management/cli/ratelimit/update-namespace` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit update-namespace --namespace=rlns_1234abcd --name=email.outbound",
			"unkey api ratelimit update-namespace --namespace=email.sends --name=email.outbound",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("namespace", "The namespace ID or current name to rename.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("name", "New case-sensitive namespace name, unique within your workspace.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Ratelimit.UpdateNamespace, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RatelimitUpdateNamespaceResponseBody)
			}

			req := components.V2RatelimitUpdateNamespaceRequestBody{
				Namespace: cmd.String("namespace"),
				Name:      cmd.String("name"),
			}
			res, err := client.Ratelimit.UpdateNamespace(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2RatelimitUpdateNamespaceResponseBody)
		},
	}
}
