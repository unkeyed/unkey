package portal

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/cli"
	"strings"
	"testing"
)

func TestCreateSession(t *testing.T) {
	tests := []struct {
		name, args string
		preview    bool
		count      int
		returnURL  *string
	}{{"minimal", "portal create-session --portal=my-portal --external-id=u --scopes=keys:read", false, 1, nil}, {"all flags", "portal create-session --portal=my-portal --external-id=u --scopes=keys:read,keys:reroll --preview=true --return-url=https://app.example.com/settings", true, 2, func() *string { v := "https://app.example.com/settings"; return &v }()}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[components.V2PortalCreateSessionRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.preview, *got.Preview)
			require.Len(t, got.Scopes, tt.count)
			require.Equal(t, tt.returnURL, got.ReturnURL)
		})
	}
}

func TestCreateSessionPermissionValidation(t *testing.T) {
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), strings.Fields("unkey portal create-session --portal=my-portal --external-id=u --scopes=keys:read,keys:delete --root-key=test"))
	require.ErrorContains(t, err, `invalid scope "keys:delete"`)
}

// keys:create and analytics:read are in the SDK's Scope enum but not in the
// portal's vocabulary, so the CLI has to reject them locally rather than let the
// caller spend a request discovering the API refuses them.
func TestValidatePortalScopes(t *testing.T) {
	t.Run("accepts the delivered scopes", func(t *testing.T) {
		require.NoError(t, validatePortalScopes("keys:read"))
		require.NoError(t, validatePortalScopes("keys:read,keys:reroll"))
	})

	// The portal reaches rerolling from the keys page, so reroll alone mints a
	// session with no page the end user can open.
	t.Run("rejects reroll without read", func(t *testing.T) {
		err := validatePortalScopes("keys:reroll")
		require.ErrorContains(t, err, "keys:reroll")
		require.ErrorContains(t, err, "keys:read")
	})

	for _, scope := range []string{"analytics:read", "keys:create"} {
		t.Run("rejects "+scope, func(t *testing.T) {
			err := validatePortalScopes(scope)
			require.ErrorContains(t, err, scope)
			require.ErrorContains(t, err, "valid choices: keys:read, keys:reroll",
				"the error must offer only the scopes that still work")
		})

		t.Run("rejects "+scope+" alongside a delivered scope", func(t *testing.T) {
			require.ErrorContains(t, validatePortalScopes("keys:read,"+scope), scope)
		})
	}
}
