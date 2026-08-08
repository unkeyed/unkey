package identities

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getIdentityCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-identity",
		Usage: "Retrieve an identity by external ID",
		Description: `Retrieve an identity by external ID. Returns metadata, rate limits, and other associated data.

Use this to check if an identity exists, view configurations, or build management dashboards.

Required permissions:
- identity.*.read_identity

For full documentation, see https://www.unkey.com/docs/api-reference/v2/identities/get-identity` + util.Disclaimer,
		Examples: []string{
			"unkey api identities get-identity --identity=user_123",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("identity", "The ID of the identity to retrieve, either externalId or identityId.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Identities.GetIdentity, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2IdentitiesGetIdentityResponseBody)
			}

			res, err := client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{
				Identity: cmd.String("identity"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2IdentitiesGetIdentityResponseBody)
		},
	}
}
