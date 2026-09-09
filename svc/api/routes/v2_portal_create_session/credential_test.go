package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	createportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
	deleteportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
	getportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
	updateportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// dashboardAdminPrincipal is the principal a WorkOS admin's dashboard token
// produces: a JWT credential whose only grant is the workspace-wide admin URN.
//
// That grant satisfies the portal session URN stage 1 evaluates, so nothing in
// the permission vocabulary keeps this principal from minting. The credential
// check is what does.
func dashboardAdminPrincipal(workspaceID string) *authprincipal.Principal {
	return &authprincipal.Principal{
		Version: authprincipal.Version,
		Subject: authprincipal.Subject{
			ID:   "user_dashboard_admin",
			Name: "dashboard admin",
			Type: authprincipal.SubjectTypeUser,
		},
		Type:                  authprincipal.TypeJWT,
		Source:                authprincipal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           []string{fmt.Sprintf("unkey:v1:%s:**#*", workspaceID)},
	}
}

// withPrincipal authenticates every request on the stack as the given principal.
//
// The harness's protected stack resolves bearer root keys only, mirroring
// production's root-key resolver, so a dashboard JWT principal cannot be
// produced by sending a header. Injecting it is the only way to exercise the
// credential type this route has to refuse.
func withPrincipal(p *authprincipal.Principal) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			s.SetPrincipal(p)
			return next(ctx, s)
		}
	}
}

// principalMiddleware is the harness's unauthenticated stack with a fixed
// principal appended, so validation and error rendering still match production.
func principalMiddleware(h *testutil.Harness, p *authprincipal.Principal) []zen.Middleware {
	stack := append([]zen.Middleware{}, h.PublicMiddleware()...)
	return append(stack, withPrincipal(p))
}

// jwtHeaders carry a bearer token the injected principal stands in for. The
// spec's security requirement is validated before any handler runs, so the
// header has to be present even though nothing resolves it here.
func jwtHeaders() http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer dashboard_jwt"},
	}
}

// TestCreateSessionRefusesDashboardTokens pins the control that replaced an
// accident. Before stage 1 evaluated URNs, a dashboard token was refused only
// because the two permission vocabularies did not meet; the admin grant covers
// the session URN, so from here on the credential type is the whole defence.
//
// A dashboard admin who could mint would receive a URL that authenticates as an
// arbitrary end user, which is impersonation rather than administration.
func TestCreateSessionRefusesDashboardTokens(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}

	workspaceID := h.Resources().UserWorkspace.ID
	h.Register(route, principalMiddleware(h, dashboardAdminPrincipal(workspaceID))...)

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	insertKeyspacePortal(t, h, workspaceID, "jwt-portal", api.KeyAuthID.String)

	t.Run("an existing portal", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, jwtHeaders(), handler.Request{
			Portal:     "jwt-portal",
			ExternalId: externalID,
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
		})
		require.Equal(t, http.StatusForbidden, res.Status, "got: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Detail, "root key",
			"the refusal must name the credential requirement, not a missing permission")

		require.Zero(t, countPortalSessions(t, h, workspaceID, externalID),
			"a refused credential must write no session row")
		require.Zero(t, countAuditEntriesMentioning(t, h, workspaceID, externalID),
			"a refused credential must write no audit entry")
	})

	// Behind the portal lookup this same call would answer 404 for an absent
	// portal and 403 for a present one, which turns the route into an existence
	// and slug oracle for any authenticated dashboard user.
	t.Run("a portal that does not exist", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, jwtHeaders(), handler.Request{
			Portal:     "no-such-portal",
			ExternalId: uid.New(uid.TestPrefix),
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
		})
		require.Equal(t, http.StatusForbidden, res.Status,
			"the credential check must precede the portal lookup: %s", res.RawBody)
	})
}

// TestDashboardTokensStillManagePortals is the other half of the restriction.
// Refusing dashboard tokens at the mint must not cost the dashboard its portal
// administration, which runs on the same credential.
func TestDashboardTokensStillManagePortals(t *testing.T) {
	h := testutil.NewHarness(t)

	workspaceID := h.Resources().UserWorkspace.ID
	stack := principalMiddleware(h, dashboardAdminPrincipal(workspaceID))

	create := &createportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	get := &getportal.Handler{DB: h.DB}
	update := &updateportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	remove := &deleteportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	for _, route := range []zen.Route{create, get, update, remove} {
		h.Register(route, stack...)
	}

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})

	createRes := testutil.CallRoute[createportal.Request, createportal.Response](h, create, jwtHeaders(), createportal.Request{
		Slug:        "managed-portal",
		DisplayName: "Managed",
		KeyspaceId:  ptr.P(openapi.PortalKeyspaceId(api.KeyAuthID.String)),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusOK, createRes.Status, "got: %s", createRes.RawBody)
	portalID := createRes.Body.Data.PortalId

	getRes := testutil.CallRoute[getportal.Request, getportal.Response](h, get, jwtHeaders(), getportal.Request{
		Portal: ptr.P(openapi.ResourceIdentifier(portalID)),
	})
	require.Equal(t, http.StatusOK, getRes.Status, "got: %s", getRes.RawBody)

	updateRes := testutil.CallRoute[updateportal.Request, updateportal.Response](h, update, jwtHeaders(), updateportal.Request{
		Portal:      openapi.ResourceIdentifier(portalID),
		DisplayName: ptr.P("Renamed"),
	})
	require.Equal(t, http.StatusOK, updateRes.Status, "got: %s", updateRes.RawBody)

	deleteRes := testutil.CallRoute[deleteportal.Request, deleteportal.Response](h, remove, jwtHeaders(), deleteportal.Request{
		Portal: openapi.ResourceIdentifier(portalID),
	})
	require.Equal(t, http.StatusOK, deleteRes.Status, "got: %s", deleteRes.RawBody)
}

// TestCreateSessionRefusesNonRootApiKey guards the two-hop guarantee the
// credential switch relies on: it admits principal.TypeAPIKey, which is only a
// root key because the resolver producing that type rejects everything else.
// The linter cannot see that hop, so it is pinned here.
func TestCreateSessionRefusesNonRootApiKey(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	insertKeyspacePortal(t, h, workspaceID, "non-root-portal", api.KeyAuthID.String)

	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspaceID,
		KeySpaceID:  api.KeyAuthID.String,
		Disabled:    false,
	})

	externalID := uid.New(uid.TestPrefix)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", key.Key)},
	}

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, handler.Request{
		Portal:     "non-root-portal",
		ExternalId: externalID,
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
	})
	require.Equal(t, http.StatusUnauthorized, res.Status,
		"a workspace key is not a root key and must not authenticate here: %s", res.RawBody)
	require.Zero(t, countPortalSessions(t, h, workspaceID, externalID))
	require.Zero(t, countAuditEntriesMentioning(t, h, workspaceID, externalID))
}
