package caller

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
)

const origin = "https://mcp.unkey.com"

func TestParseHeaderPrecedence(t *testing.T) {
	rootKey := uid.New(uid.KeyPrefix)
	other := uid.New(uid.KeyPrefix)

	cases := []struct {
		name   string
		header http.Header
		want   string
	}{
		{name: "authorization bearer", header: http.Header{"Authorization": {"Bearer " + rootKey}}, want: rootKey},
		{name: "lowercase scheme", header: http.Header{"Authorization": {"bearer " + rootKey}}, want: rootKey},
		{name: "speakeasy bearer header", header: http.Header{"Mcp-Unkey-V2-Bearer": {rootKey}}, want: rootKey},
		{name: "speakeasy bearer header with scheme", header: http.Header{"Mcp-Unkey-V2-Bearer": {"Bearer " + rootKey}}, want: rootKey},
		{name: "speakeasy root key header", header: http.Header{"Mcp-Unkey-V2-Root-Key": {rootKey}}, want: rootKey},
		{name: "authorization wins", header: http.Header{"Authorization": {"Bearer " + rootKey}, "Mcp-Unkey-V2-Bearer": {other}}, want: rootKey},
		{name: "empty authorization falls through", header: http.Header{"Authorization": {" "}, "Mcp-Unkey-V2-Root-Key": {rootKey}}, want: rootKey},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			credential, err := Parse(tc.header)
			require.NoError(t, err)
			require.Equal(t, KindRootKey, credential.Kind())
			require.Equal(t, "Bearer "+tc.want, credential.Authorization())
		})
	}
}

func TestParseRejects(t *testing.T) {
	signer := newSigner(t)
	notSession := sign(t, signer, clisession.Claims{TokenUse: "dashboard"})

	_, err := Parse(http.Header{})
	require.ErrorIs(t, err, ErrMissing)

	_, err = Parse(http.Header{"Authorization": {"Basic " + uid.New(uid.KeyPrefix)}})
	require.ErrorIs(t, err, ErrUnusable)

	_, err = Parse(http.Header{"Authorization": {"Bearer " + notSession}})
	require.ErrorIs(t, err, ErrUnusable)
}

// TestTriage guarantees that svc/mcp answers 401 for session tokens the
// client must refresh or replace, which is the only signal that makes an
// OAuth client do so, and never second-guesses root keys.
func TestTriage(t *testing.T) {
	signer := newSigner(t)
	now := time.Now()
	session := func(expiresAt time.Time, audience ...string) Credential {
		t.Helper()
		credential, err := Parse(http.Header{"Authorization": {"Bearer " + sign(t, signer, clisession.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Audience: audience, ExpiresAt: expiresAt.Unix()},
			TokenUse:         clisession.TokenUse,
		})}})
		require.NoError(t, err)
		require.Equal(t, KindOAuth, credential.Kind())
		return credential
	}

	require.NoError(t, session(now.Add(time.Minute), clisession.Audience, origin).Triage(now, origin))
	require.NoError(t, session(now.Add(time.Minute), clisession.Audience, origin+"/compute").Triage(now, origin), "any resource on the origin")
	require.NoError(t, session(now.Add(time.Minute), "HTTPS://MCP.UNKEY.COM/mcp/v2").Triage(now, origin), "origin comparison ignores case")

	require.ErrorIs(t, session(now.Add(29*time.Second), origin).Triage(now, origin), ErrUnusable, "expiring within the margin")
	require.ErrorIs(t, session(now.Add(-time.Second), origin).Triage(now, origin), ErrUnusable, "expired")
	require.ErrorIs(t, session(now.Add(time.Minute), clisession.Audience).Triage(now, origin), ErrUnusable, "CLI token")
	require.ErrorIs(t, session(now.Add(time.Minute), "https://mcp.unkey.com.evil.example").Triage(now, origin), ErrUnusable, "other origin")

	rootKey, err := Parse(http.Header{"Authorization": {"Bearer " + uid.New(uid.KeyPrefix)}})
	require.NoError(t, err)
	require.NoError(t, rootKey.Triage(now, origin))
}

// TestWorkspaceIDRequiresAcceptedToken guarantees that an unverified
// workspace claim reaches analytics only after the API accepted the token,
// so a forged token cannot attribute calls to another workspace.
func TestWorkspaceIDRequiresAcceptedToken(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	sessionID := uid.New(uid.CLISessionPrefix)
	credential, err := Parse(http.Header{"Authorization": {"Bearer " + sign(t, newSigner(t), clisession.Claims{
		WorkspaceID: workspaceID,
		SessionID:   sessionID,
		TokenUse:    clisession.TokenUse,
	})}})
	require.NoError(t, err)

	require.Equal(t, workspaceID, credential.WorkspaceID(http.StatusOK))
	require.Empty(t, credential.WorkspaceID(http.StatusBadRequest))
	require.Empty(t, credential.WorkspaceID(http.StatusUnauthorized))
	require.Equal(t, sessionID, credential.SessionID())

	rootKey, err := Parse(http.Header{"Authorization": {"Bearer " + uid.New(uid.KeyPrefix)}})
	require.NoError(t, err)
	require.Empty(t, rootKey.WorkspaceID(http.StatusOK))
}

func newSigner(t *testing.T) *jwt.HS256Signer[clisession.Claims] {
	t.Helper()
	signer, err := jwt.NewHS256Signer[clisession.Claims]([]byte(uid.New(uid.TestPrefix, 32)))
	require.NoError(t, err)
	return signer
}

func sign(t *testing.T, signer *jwt.HS256Signer[clisession.Claims], claims clisession.Claims) string {
	t.Helper()
	token, err := signer.Sign(claims)
	require.NoError(t, err)
	return token
}
