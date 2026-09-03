package portal

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// portalScopes is the vocabulary createSession still accepts. The SDK's Scope
// enum is wider: keys:create and analytics:read have no portal feature behind
// them and the API rejects them, so offering them here would only produce a 400
// the caller cannot act on.
var portalScopes = []string{
	string(components.ScopeKeysRead), string(components.ScopeKeysReroll),
}

func validatePortalScopes(value string) error {
	var scopes []string
	for _, scope := range strings.Split(value, ",") {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if !slices.Contains(portalScopes, scope) {
			return fmt.Errorf("invalid scope %q; valid choices: %s", scope, strings.Join(portalScopes, ", "))
		}
		scopes = append(scopes, scope)
	}

	// Mirrors the API: the portal reaches rerolling from the keys page, which
	// renders only for keys:read, so reroll alone mints a session with no page
	// the end user can open. Caught here so the failure lands on the command
	// rather than a round trip later.
	if slices.Contains(scopes, "keys:reroll") && !slices.Contains(scopes, "keys:read") {
		return fmt.Errorf("scope %q requires %q in the same session", "keys:reroll", "keys:read")
	}

	return nil
}

func createSessionCmd() *cli.Command {
	return &cli.Command{Name: "create-session", Usage: "Create a short-lived session token for an end user to access the Customer Portal", Description: `Create a short-lived session token for an end user to access the Customer Portal.

The returned exchange code is valid for 15 minutes and can be exchanged exactly once for a 24-hour portal access token via portal.exchangeCode. Redirect the end user to the returned URL to start the portal experience.

Required Permissions

Your root key must be associated with a workspace that has an enabled portal configuration.

` + util.Disclaimer,
		Examples: []string{"unkey api portal create-session --portal=my-portal --external-id=user_123 --scopes=keys:read,keys:reroll", "unkey api portal create-session --portal=my-portal --external-id=user_123 --scopes=keys:read --return-url=https://app.example.com/settings/api-keys"},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("portal", "Portal configuration ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("external-id", "End user's identifier in your system.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.StringSlice("scopes", "Portal capabilities. Valid choices: "+strings.Join(portalScopes, ", ")+".", cli.Required(), cli.Validate(validatePortalScopes), cli.MutuallyExclusive("body")), cli.Bool("preview", "Create a preview session.", cli.Default(false), cli.MutuallyExclusive("body")), cli.String("return-url", "Absolute URL to return the end user to after the portal.", cli.MutuallyExclusive("body"))},
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
			values := cmd.StringSlice("scopes")
			scopes := make([]components.Scope, len(values))
			for i, value := range values {
				scopes[i] = components.Scope(value)
			}
			req := components.V2PortalCreateSessionRequestBody{Portal: cmd.String("portal"), ExternalID: cmd.String("external-id"), Scopes: scopes, Preview: ptr.P(cmd.Bool("preview")), ReturnURL: nil}
			if v := cmd.String("return-url"); v != "" {
				req.ReturnURL = &v
			}
			res, err := client.Portal.CreateSession(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PortalCreateSessionResponseBody)
		},
	}
}
