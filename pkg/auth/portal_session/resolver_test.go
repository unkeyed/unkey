package portalsession

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/portal"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

type stubPortal struct {
	token string
	info  *portal.SessionInfo
}

func (s stubPortal) GetSession(_ context.Context, token string) (*portal.SessionInfo, error) {
	if token != s.token {
		return nil, errors.New("invalid portal session")
	}
	return s.info, nil
}

func TestResolver_ResolvePortalCookie(t *testing.T) {
	t.Parallel()

	// A valid portal cookie must resolve into a user principal scoped to the session workspace.
	resolver := NewResolver(stubPortal{
		token: "portal_session_123",
		info: &portal.SessionInfo{
			WorkspaceID:    "ws_123",
			ExternalID:     "customer_123",
			PortalConfigID: "pc_123",
			Permissions:    []string{"api.*.read_key"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "portal_session_123"})
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0, true))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, authprincipal.Version, principal.Version)
	require.Equal(t, authprincipal.TypePortalSession, principal.Type)
	require.Equal(t, "customer_123", principal.Subject.ID)
	require.Equal(t, "ws_123", principal.WorkspaceID)
	source, ok := principal.Source.(authprincipal.PortalSessionSource)
	require.True(t, ok)
	require.Equal(t, "portal_session_123", source.SessionID)
	require.Equal(t, "pc_123", source.PortalConfigID)
	require.Equal(t, []string{"api.*.read_key"}, source.Permissions)
	require.True(t, rbac.HasAnyPermission(principal.Permissions, rbac.Api, rbac.ReadKey))
}

func TestResolver_IgnoresMissingCookie(t *testing.T) {
	t.Parallel()

	// Missing portal cookies must yield to the next configured credential resolver.
	resolver := NewResolver(stubPortal{})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0, true))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Nil(t, principal)
}

func TestResolver_IgnoresCookieWhenBearerIsPresent(t *testing.T) {
	t.Parallel()

	// Explicit bearer credentials must take precedence over a browser portal cookie.
	resolver := NewResolver(stubPortal{
		token: "portal_session_123",
		info: &portal.SessionInfo{
			WorkspaceID: "ws_123",
			ExternalID:  "customer_123",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer root_key")
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "portal_session_123"})
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0, true))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Nil(t, principal)
}
