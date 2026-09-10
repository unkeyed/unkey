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

// This file pins the inverse of what most urn_permissions_test.go files assert.
// Elsewhere a canonical URN grant is expected to authorize a route. Portal
// session minting evaluates legacy tuples only, so a URN grant -- including the
// global permission produced by the JWT admin role -- is denied here.
//
// That is a deliberate, accepted consequence rather than an oversight: a
// separate migration moves all of authorization to URN-only, and half-converting
// it from this route would leave call sites in two different worlds. The
// practical effect is that no dashboard JWT principal can mint a portal session
// until that migration lands.
//
// These tests are the signal for that migration. When URN arms are added here
// they will fail, and each expectation flips from denied to allowed.

// urnDenialFixture seeds a keyspace-mapped portal, under the given slug, whose
// keyspace has an owning API. Stage 2 needs that API to express its API-scoped
// checks, so a portal without one would fail for the wrong reason. It returns
// the project and keyspace IDs for canonical URN grants.
func urnDenialFixture(t *testing.T, h *testutil.Harness, slug string) (string, string) {
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
	insertKeyspacePortal(t, h, workspaceID, slug, api.KeyAuthID.String)

	return api.ProjectID, api.KeyAuthID.String
}

func TestCreateSessionDeniesURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID

	t.Run("admin grant cannot mint", func(t *testing.T) {
		slug := "urn-admin-portal"
		_, _ = urnDenialFixture(t, h, slug)

		// This is exactly what the JWT admin role expands to, so this case
		// also asserts that dashboard JWT principals cannot mint.
		rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:**#*", workspaceID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
			Portal:     slug,
			ExternalId: uid.New(uid.TestPrefix),
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
		})
		require.Equal(t, http.StatusNotFound, res.Status, "a stage-1 denial is masked as 404, got: %s", res.RawBody)
	})

	t.Run("canonical URN grants cannot mint", func(t *testing.T) {
		slug := "urn-portal-portal"
		projectID, keyspaceID := urnDenialFixture(t, h, slug)

		// Valid project-scoped URN grants do not satisfy the legacy portal tuple.
		// Stage 1 refuses them first.
		rootKey := h.CreateRootKey(workspaceID,
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
		require.Equal(t, http.StatusNotFound, res.Status, "a stage-1 denial is masked as 404, got: %s", res.RawBody)
	})

	t.Run("legacy portal tuple with URN-only scope grant is refused at stage 2", func(t *testing.T) {
		slug := "urn-mixed-portal"
		projectID, keyspaceID := urnDenialFixture(t, h, slug)

		// Stage 1 passes on the legacy tuple, so this isolates stage 2 and shows
		// the two grant forms are not interchangeable there either.
		rootKey := h.CreateRootKey(workspaceID,
			"portal.*.create_portal_session",
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#read", workspaceID, projectID, keyspaceID),
			fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#read", workspaceID, projectID, keyspaceID),
		)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{
			Portal:     slug,
			ExternalId: uid.New(uid.TestPrefix),
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
		})
		require.Equal(t, http.StatusForbidden, res.Status, "got: %s", res.RawBody)
	})
}
