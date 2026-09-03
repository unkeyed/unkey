package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

// urnFixture seeds a keyspace-mapped portal, under the given slug, whose
// keyspace has an owning API. Stage 2 needs that API to express its API-scoped
// checks, so a portal without one would fail for the wrong reason. It returns
// the portal, project, and keyspace IDs for canonical URN grants.
func urnFixture(t *testing.T, h *testutil.Harness, slug string) (string, string, string) {
	t.Helper()

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspaceID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	portalID := insertKeyspacePortal(t, h, workspaceID, slug, api.KeyAuthID.String)

	return portalID, api.ProjectID, api.KeyAuthID.String
}

func TestCreateSessionAuthorizesURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID

	t.Run("admin grant", func(t *testing.T) {
		slug := "urn-admin-portal"
		_, _, _ = urnFixture(t, h, slug)

		rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:**#*", workspaceID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Portal:     slug,
			ExternalId: uid.New(uid.TestPrefix),
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
		})
		require.Equal(t, http.StatusOK, res.Status, "the admin grant must authorize session creation: %s", res.RawBody)
	})

	t.Run("project portal session grant", func(t *testing.T) {
		slug := "urn-portal-portal"
		portalID, projectID, keyspaceID := urnFixture(t, h, slug)

		rootKey := h.CreateRootKey(workspaceID,
			fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/sessions/*#write", workspaceID, projectID, portalID),
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#read", workspaceID, projectID, keyspaceID),
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#read", workspaceID, projectID, keyspaceID),
		)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Portal:     slug,
			ExternalId: uid.New(uid.TestPrefix),
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
		})
		require.Equal(t, http.StatusOK, res.Status, "the project portal session grant must authorize creation: %s", res.RawBody)
	})

	t.Run("another project portal session grant", func(t *testing.T) {
		slug := "wrong-project-portal"
		portalID, projectID, keyspaceID := urnFixture(t, h, slug)

		rootKey := h.CreateRootKey(workspaceID,
			fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/sessions/*#write", workspaceID, uid.New(uid.ProjectPrefix), portalID),
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#read", workspaceID, projectID, keyspaceID),
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#read", workspaceID, projectID, keyspaceID),
		)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
			Portal:     slug,
			ExternalId: uid.New(uid.TestPrefix),
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
		})
		require.Equal(t, http.StatusNotFound, res.Status, "a grant for another project must not authorize session creation: %s", res.RawBody)
	})

	t.Run("analytics remains legacy-only", func(t *testing.T) {
		slug := "urn-mixed-portal"
		_, projectID, keyspaceID := urnFixture(t, h, slug)

		// The analytics endpoint still reads legacy tuples. Do not let a keyspace
		// log URN grant mint a capability that the same caller cannot use.
		rootKey := h.CreateRootKey(workspaceID,
			"portal.*.create_portal_session",
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/logs#read", workspaceID, projectID, keyspaceID),
		)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{
			Portal:     slug,
			ExternalId: uid.New(uid.TestPrefix),
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.AnalyticsRead},
		})
		require.Equal(t, http.StatusForbidden, res.Status, "got: %s", res.RawBody)
	})
}
