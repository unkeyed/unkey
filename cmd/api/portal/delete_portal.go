package portal

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deletePortalCmd() *cli.Command {
	return &cli.Command{Name: "delete-portal", Usage: "Delete a portal and revoke the sessions it minted.", Description: `Delete a portal and revoke the sessions it minted.

Unreleased and subject to change without notice.

Its end users lose access rather than keeping it until their tokens expire. Revocation is not instantaneous: session lookups are cached briefly, so a request already in flight may still succeed.

The app or keyspace it served is untouched, and its slug becomes free for a new portal.

Required Permissions

Your root key must have one of:
- portal.*.delete_portal (to delete any portal in the workspace)
- portal.<portal_id>.delete_portal (to delete a specific portal)

Without the permission this returns 404, not 403.

For full documentation, see https://www.unkey.com/docs/api-reference/portal/delete-portal` + util.Disclaimer, Examples: []string{"unkey api portal delete-portal --portal=acme-portal"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("portal", "Portal ID or slug.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}
		if cmd.FlagIsSet("body") {
			res, err := util.SendBody(ctx, client.Portal.DeletePortal, cmd.String("body"))
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2PortalDeletePortalResponseBody)
		}
		res, err := client.Portal.DeletePortal(ctx, components.V2PortalDeletePortalRequestBody{Portal: cmd.String("portal")})
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2PortalDeletePortalResponseBody)
	}}
}
