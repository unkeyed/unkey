package portal

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func createPortalCmd() *cli.Command {
	return &cli.Command{Name: "create-portal", Usage: "Create a portal for one app or keyspace in your workspace.", Description: `Create a portal for one app or keyspace in your workspace.

Unreleased and subject to change without notice.

Send exactly one of keyspaceId or appId. That resource must belong to your workspace, and it can back only one portal, so a second portal for the same resource is a 409.

displayName is what your end users see. It is yours to set and change independently of the resource the portal serves.

Required Permissions

Your root key must have portal.*.create_portal. A grant scoped to a specific portal id does not authorize creation, because the id does not exist yet.

For full documentation, see https://www.unkey.com/docs/api-reference/portal/create-portal` + util.Disclaimer,
		Examples:     []string{"unkey api portal create-portal --slug=acme-portal --display-name=Acme --keyspace-id=ks_1234abcd", "unkey api portal create-portal --slug=developer-portal --display-name='Developer Portal' --app-id=app_1234abcd --logo-url=https://cdn.example.com/logo.svg --primary-color=#6366f1"},
		Flags:        []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("slug", "URL-safe portal handle unique within your workspace.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("display-name", "Human-readable name shown to end users.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("keyspace-id", "ID of the keyspace this portal serves.", cli.MutuallyExclusive("body")), cli.String("app-id", "ID of the app this portal serves.", cli.MutuallyExclusive("body")), cli.Bool("enabled", "Allow sessions to be minted for this portal.", cli.Default(true), cli.MutuallyExclusive("body")), cli.String("logo-url", "Absolute HTTPS URL of the portal logo.", cli.MutuallyExclusive("body")), cli.String("primary-color", "Six-digit hex colour for primary actions.", cli.MutuallyExclusive("body"))},
		RequireOneOf: [][]string{{"keyspace-id", "app-id"}},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}
			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Portal.CreatePortal, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PortalCreatePortalResponseBody)
			}

			slug := cmd.String("slug")
			displayName := cmd.String("display-name")
			enabled := ptr.P(cmd.Bool("enabled"))
			var logoURL *string
			var primaryColor *string
			if v := cmd.String("logo-url"); v != "" {
				logoURL = &v
			}
			if v := cmd.String("primary-color"); v != "" {
				primaryColor = &v
			}
			var req components.V2PortalCreatePortalRequestBodyUnion
			if keyspaceID := cmd.String("keyspace-id"); keyspaceID != "" {
				req = components.CreateV2PortalCreatePortalRequestBodyUnionV2PortalCreatePortalRequestBody1(components.V2PortalCreatePortalRequestBody1{Slug: slug, DisplayName: displayName, KeyspaceID: keyspaceID, AppID: nil, Enabled: enabled, LogoURL: logoURL, PrimaryColor: primaryColor})
			} else {
				req = components.CreateV2PortalCreatePortalRequestBodyUnionV2PortalCreatePortalRequestBody2(components.V2PortalCreatePortalRequestBody2{Slug: slug, DisplayName: displayName, KeyspaceID: nil, AppID: cmd.String("app-id"), Enabled: enabled, LogoURL: logoURL, PrimaryColor: primaryColor})
			}
			res, err := client.Portal.CreatePortal(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PortalCreatePortalResponseBody)
		},
	}
}
