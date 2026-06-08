package zen

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
)

// TestSession_GetPrincipalRequiresAuthentication verifies handlers fail closed
// when a principal is missing. This is the guardrail for accidentally registering
// a protected handler without authentication middleware: the handler can return
// this error directly instead of continuing as an anonymous request.
func TestSession_GetPrincipalRequiresAuthentication(t *testing.T) {
	t.Parallel()

	sess := &Session{}

	p, err := sess.GetPrincipal()

	require.Error(t, err)
	require.Nil(t, p)
}

// TestSession_PrincipalScopesWorkspaceMetadata verifies request-wide workspace
// metadata is derived from the authenticated principal. Metrics and error logs
// still need a workspace ID, but storing only the principal avoids a second
// mutable source of truth that could diverge from authorization scope.
func TestSession_PrincipalScopesWorkspaceMetadata(t *testing.T) {
	t.Parallel()

	want := &principal.Principal{
		Version:     principal.Version,
		Subject:     principal.Subject{ID: "root_key_123", Name: "Root Key", Type: principal.SubjectTypeRootKey},
		Type:        principal.TypeAPIKey,
		Source:      principal.KeySource{KeyID: "key_123", KeySpaceID: "ks_123"},
		WorkspaceID: "ws_123",
		Permissions: []string{"api.*.read_key"},
	}
	sess := &Session{}

	sess.SetPrincipal(want)
	got, err := sess.GetPrincipal()

	require.NoError(t, err)
	require.Same(t, want, got)
}

// TestSession_ResetClearsPrincipal verifies pooled sessions cannot leak an
// authenticated principal into a later request. Session reuse is an auth-critical
// boundary because stale principals would turn one successful request into
// implicit authentication for the next request using the same pooled object.
func TestSession_ResetClearsPrincipal(t *testing.T) {
	t.Parallel()

	sess := &Session{}
	sess.SetPrincipal(&principal.Principal{
		Version:     principal.Version,
		Subject:     principal.Subject{ID: "root_key_123", Name: "Root Key", Type: principal.SubjectTypeRootKey},
		Type:        principal.TypeAPIKey,
		Source:      principal.KeySource{KeyID: "key_123", KeySpaceID: "ks_123"},
		WorkspaceID: "ws_123",
		Permissions: []string{"api.*.read_key"},
	})

	sess.reset()

	got, err := sess.GetPrincipal()
	require.Error(t, err)
	require.Nil(t, got)
}
