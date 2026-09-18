package portal

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getPortalCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-portal",
		Usage: "Read one portal, by its id or slug, or by the resource it serves.",
		Description: `Read one portal, by its id or slug, or by the resource it serves.

Unreleased and subject to change without notice.

Send exactly one of portal, keyspaceId, or appId. Sending more than one is a 400.

Required Permissions

Your root key must have one of:
- portal.*.read_portal (to read any portal in the workspace)
- portal.<portal_id>.read_portal (to read a specific portal)

Without the permission this returns 404, not 403.

For full documentation, see https://www.unkey.com/docs/api-reference/portal/get-portal` + util.Disclaimer,
		Examples: []string{"unkey api portal get-portal --portal=acme-portal", "unkey api portal get-portal --keyspace-id=ks_1234abcd", "unkey api portal get-portal --app-id=app_1234abcd"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("portal", "Portal ID or slug.", cli.MutuallyExclusive("body")), cli.String("keyspace-id", "ID of the keyspace served by the portal.", cli.MutuallyExclusive("body")), cli.String("app-id", "ID of the app served by the portal.", cli.MutuallyExclusive("body"))},
		RequireOneOf: [][]string{{
			"portal",
			"keyspace-id",
			"app-id",
		}},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}
			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Portal.GetPortal, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PortalGetPortalResponseBody)
			}
			var req components.V2PortalGetPortalRequestBodyUnion
			if portal := cmd.String("portal"); portal != "" {
				req = components.CreateV2PortalGetPortalRequestBodyUnionV2PortalGetPortalRequestBody1(components.V2PortalGetPortalRequestBody1{
					Portal:     portal,
					KeyspaceID: nil,
					AppID:      nil,
				})
			} else if keyspaceID := cmd.String("keyspace-id"); keyspaceID != "" {
				req = components.CreateV2PortalGetPortalRequestBodyUnionV2PortalGetPortalRequestBody2(components.V2PortalGetPortalRequestBody2{
					Portal:     nil,
					KeyspaceID: keyspaceID,
					AppID:      nil,
				})
			} else {
				req = components.CreateV2PortalGetPortalRequestBodyUnionV2PortalGetPortalRequestBody3(components.V2PortalGetPortalRequestBody3{
					Portal:     nil,
					KeyspaceID: nil,
					AppID:      cmd.String("app-id"),
				})
			}
			res, err := client.Portal.GetPortal(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PortalGetPortalResponseBody)
		},
	}
}
