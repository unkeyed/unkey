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

// keys:create is still in the SDK's Scope enum, so the CLI has to reject it locally.
func TestValidatePortalScopes(t *testing.T) {
	t.Run("accepts the delivered scopes", func(t *testing.T) {
		require.NoError(t, validatePortalScopes("keys:read"))
		require.NoError(t, validatePortalScopes("keys:read,keys:reroll"))
		require.NoError(t, validatePortalScopes("keys:read,analytics:read"))
	})

	// Both are reached from the keys page, so neither can stand alone.
	for _, scope := range []string{"keys:reroll", "analytics:read"} {
		t.Run("rejects "+scope+" without read", func(t *testing.T) {
			err := validatePortalScopes(scope)
			require.ErrorContains(t, err, scope)
			require.ErrorContains(t, err, "keys:read")
		})
	}

	t.Run("rejects keys:create", func(t *testing.T) {
		err := validatePortalScopes("keys:create")
		require.ErrorContains(t, err, "keys:create")
		require.ErrorContains(t, err, "valid choices: keys:read, keys:reroll, analytics:read",
			"the error must offer only the scopes that still work")
	})

	t.Run("rejects keys:create alongside a delivered scope", func(t *testing.T) {
		require.ErrorContains(t, validatePortalScopes("keys:read,keys:create"), "keys:create")
	})
}
