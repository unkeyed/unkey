package portal

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

var portalPermissions = []string{
	string(components.PermissionEnumKeysRead), string(components.PermissionEnumKeysCreate),
	string(components.PermissionEnumKeysReroll), string(components.PermissionEnumAnalyticsRead),
}

func validatePortalPermissions(value string) error {
	for _, permission := range strings.Split(value, ",") {
		permission = strings.TrimSpace(permission)
		if permission != "" && !slices.Contains(portalPermissions, permission) {
			return fmt.Errorf("invalid permission %q; valid choices: %s", permission, strings.Join(portalPermissions, ", "))
		}
	}
	return nil
}

func createSessionCmd() *cli.Command {
	return &cli.Command{Name: "create-session", Usage: "Create a short-lived session token for an end user to access the Customer Portal", Description: `Create a short-lived session token for an end user to access the Customer Portal.

The returned session ID is valid for 15 minutes and can be exchanged exactly once for a 24-hour browser session via portal.exchangeSession. Redirect the end user to the returned URL to start the portal experience.

Required Permissions

Your root key must be associated with a workspace that has an enabled portal configuration.

` + util.Disclaimer,
		Examples: []string{"unkey api portal create-session --slug=my-portal --external-id=user_123 --permissions=keys:read,keys:reroll", "unkey api portal create-session --slug=my-portal --external-id=user_123 --permissions=analytics:read --preview=true"},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("slug", "Portal configuration slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("external-id", "End user's identifier in your system.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.StringSlice("permissions", "Portal capabilities. Valid choices: "+strings.Join(portalPermissions, ", ")+".", cli.Required(), cli.Validate(validatePortalPermissions), cli.MutuallyExclusive("body")), cli.Bool("preview", "Create a preview session.", cli.Default(false), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Portal.CreateSession, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PortalCreateSessionResponseBody)
			}
			values := cmd.StringSlice("permissions")
			permissions := make([]components.PermissionEnum, len(values))
			for i, value := range values {
				permissions[i] = components.PermissionEnum(value)
			}
			req := components.V2PortalCreateSessionRequestBody{Slug: cmd.String("slug"), ExternalID: cmd.String("external-id"), Permissions: permissions, Preview: ptr.P(cmd.Bool("preview"))}
			res, err := client.Portal.CreateSession(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PortalCreateSessionResponseBody)
		},
	}
}
