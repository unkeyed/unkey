package identities

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteIdentityCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-identity",
		Usage: "Permanently delete an identity.",
		Description: `Permanently delete an identity. This operation cannot be undone.

Use this for data cleanup, compliance requirements, or when removing entities from your system.

Important:
- Requires identity.*.delete_identity permission
- Associated API keys remain functional but lose shared resources
- External ID becomes available for reuse immediately

For full documentation, see https://www.unkey.com/docs/api-reference/v2/identities/delete-identity` + util.Disclaimer,
		Examples: []string{
			"unkey api identities delete-identity --identity=user_123",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("identity", "The identity ID or external ID to delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2IdentitiesDeleteIdentityRequestBody{Identity: cmd.String("identity")})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Permanently delete identity %q?", req.Identity)); err != nil {
				return err
			}

			res, err := client.Identities.DeleteIdentity(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2IdentitiesDeleteIdentityResponseBody)
		},
	}
}
