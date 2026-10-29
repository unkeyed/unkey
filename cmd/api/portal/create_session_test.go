package portal

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCreateSession(t *testing.T) {
	tests := []struct {
		name, args string
		count      int
		returnURL  *string
	}{{"minimal", "portal create-session --portal=my-portal --external-id=u --scopes=keys:read", 1, nil}, {"all flags", "portal create-session --portal=my-portal --external-id=u --scopes=keys:read,keys:reroll --return-url=https://app.example.com/settings", 2, func() *string { v := "https://app.example.com/settings"; return &v }()}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[components.V2PortalCreateSessionRequestBody](t, Cmd(), tt.args)
			require.Len(t, got.Scopes, tt.count)
			require.Equal(t, tt.returnURL, got.ReturnURL)
		})
	}
}

func TestCreateSessionPermissionValidation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	for _, invalid := range []string{"keys:delete", uid.New(uid.KeyPrefix)} {
		t.Run(invalid, func(t *testing.T) {
			root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
			err := root.Run(t.Context(), []string{
				"unkey", "portal", "create-session", "--portal=my-portal", "--external-id=" + uid.New(uid.IdentityPrefix),
				"--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL, "--scopes=keys:read," + invalid,
			})
			var input *cli.InputError
			require.ErrorAs(t, err, &input)
			require.Equal(t, cli.InputErrorInvalidFlagValue, input.Kind)
			require.Equal(t, []string{"scopes"}, input.Flags)
			require.ErrorIs(t, err, cli.ErrValidationFailed)
			require.ErrorContains(t, err, "Valid choices: keys:read, keys:reroll, analytics:read")
			require.NotContains(t, err.Error(), invalid)
			require.Zero(t, calls.Load())
		})
	}
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
