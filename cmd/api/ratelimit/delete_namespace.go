package ratelimit

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteNamespaceCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-namespace",
		Usage: "Delete a rate limit namespace and all of its overrides, identified by the namespace ID or name",
		Description: `Delete a rate limit namespace and all of its overrides, identified by the namespace ID or name.

Important: A deleted name cannot be reused. ratelimit.limit and ratelimit.multiLimit return 410 for it instead of creating a new namespace, so a deleted namespace does not turn into unlimited traffic. Other API nodes can take up to a minute to see the deletion.

Required permissions:
- unkey:v1:<workspace_id>:projects/*/ratelimits/namespaces/*#delete (to delete any namespace)
- unkey:v1:<workspace_id>:projects/<project_id>/ratelimits/namespaces/<namespace_id>#delete (to delete a specific namespace)

For full documentation, see https://www.unkey.com/docs/api-management/cli/ratelimit/delete-namespace` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit delete-namespace --namespace=email.outbound",
			"unkey api ratelimit delete-namespace --namespace=rlns_1234abcd --yes",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("namespace", "The namespace ID or name to delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2RatelimitDeleteNamespaceRequestBody{
				Namespace: cmd.String("namespace"),
			})
			if err != nil {
				return err
			}
			question := fmt.Sprintf("Delete rate limit namespace %q and all of its overrides? Its name cannot be reused.", req.Namespace)
			if err := util.Confirm(cmd, question); err != nil {
				return err
			}

			res, err := client.Ratelimit.DeleteNamespace(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2RatelimitDeleteNamespaceResponseBody)
		},
	}
}
