package identities

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var getIdentityCmd = &cli.Command{
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
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("identity", "The ID of the identity to retrieve, either externalId or identityId.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		res, err := client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{
			Identity: cmd.String("identity"),
		})
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2IdentitiesGetIdentityResponseBody, time.Since(start))
	},
}
