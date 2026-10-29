package ratelimit

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createNamespaceCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-namespace",
		Usage: "Create a rate limit namespace before you rate limit against it",
		Description: `Create a rate limit namespace before you rate limit against it.

ratelimit.limit also creates a namespace the first time you use its name, when your root key can create namespaces. Use this endpoint when you want the namespace to exist up front, for example to set overrides before the first request.

Important: Names are unique within your workspace, deleted namespaces included. A name that is in use or was deleted returns 409.

Required permissions:
- unkey:v1:<workspace_id>:projects/*/ratelimits/namespaces/*#write

For full documentation, see https://www.unkey.com/docs/api-management/cli/ratelimit/create-namespace` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit create-namespace --name=email.outbound",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("name", "Case-sensitive namespace name, unique within your workspace.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Ratelimit.CreateNamespace, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RatelimitCreateNamespaceResponseBody)
			}

			req := components.V2RatelimitCreateNamespaceRequestBody{
				Name: cmd.String("name"),
			}
			res, err := client.Ratelimit.CreateNamespace(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitCreateNamespaceResponseBody)
		},
	}
}
