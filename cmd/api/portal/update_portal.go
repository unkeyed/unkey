package portal

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func updatePortalCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-portal",
		Usage: "Change a portal's slug, display name, the resource it serves, its enabled state, or its branding.",
		Description: `Change a portal's slug, display name, the resource it serves, its enabled state, or its branding.

Unreleased and subject to change without notice.

Only the fields you send change. Omitting a field leaves it as it is, and for branding, sending null clears it. Send at most one of keyspaceId or appId.

Two changes affect your end users immediately:
- Re-pointing at a different resource revokes the portal's live sessions, because a session carries the scope it was minted with.
- Disabling stops new sessions but leaves live ones running until they expire.

Required Permissions

Your root key must have one of:
- portal.*.update_portal (to update any portal in the workspace)
- portal.<portal_id>.update_portal (to update a specific portal)

Without the permission this returns 404, not 403.

For full documentation, see https://www.unkey.com/docs/api-reference/portal/update-portal` + util.Disclaimer,
		Examples: []string{"unkey api portal update-portal --portal=acme-portal --display-name='Acme Developer Portal'", "unkey api portal update-portal --portal=acme-portal --app-id=app_1234abcd --enabled=false", "unkey api portal update-portal --portal=acme-portal --logo-url=null --primary-color=null"},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("portal", "Portal ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("slug", "New portal handle.", cli.MutuallyExclusive("body")), cli.String("display-name", "New human-readable name shown to end users.", cli.MutuallyExclusive("body")), cli.String("keyspace-id", "ID of the new keyspace to serve.", cli.MutuallyExclusive("body", "app-id")), cli.String("app-id", "ID of the new app to serve.", cli.MutuallyExclusive("body", "keyspace-id")), cli.Bool("enabled", "Allow new sessions to be minted.", cli.MutuallyExclusive("body")), cli.String("logo-url", "Absolute HTTPS logo URL, or null to remove it.", cli.MutuallyExclusive("body")), cli.String("primary-color", "Six-digit hex colour, or null for default styling.", cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}
			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Portal.UpdatePortal, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PortalUpdatePortalResponseBody)
			}
			req := components.V2PortalUpdatePortalRequestBody{
				Portal:       cmd.String("portal"),
				Slug:         nil,
				DisplayName:  nil,
				KeyspaceID:   nil,
				AppID:        nil,
				Enabled:      nil,
				LogoURL:      nil,
				PrimaryColor: nil,
			}
			if v := cmd.String("slug"); v != "" {
				req.Slug = &v
			}
			if v := cmd.String("display-name"); v != "" {
				req.DisplayName = &v
			}
			if v := cmd.String("keyspace-id"); v != "" {
				req.KeyspaceID = &v
			}
			if v := cmd.String("app-id"); v != "" {
				req.AppID = &v
			}
			if cmd.FlagIsSet("enabled") {
				req.Enabled = ptr.P(cmd.Bool("enabled"))
			}
			if cmd.FlagIsSet("logo-url") {
				v := cmd.String("logo-url")
				if v == "null" {
					req.LogoURL = optionalnullable.From[string](nil)
				} else {
					req.LogoURL = optionalnullable.From(&v)
				}
			}
			if cmd.FlagIsSet("primary-color") {
				v := cmd.String("primary-color")
				if v == "null" {
					req.PrimaryColor = optionalnullable.From[string](nil)
				} else {
					req.PrimaryColor = optionalnullable.From(&v)
				}
			}
			res, err := client.Portal.UpdatePortal(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PortalUpdatePortalResponseBody)
		},
	}
}
