package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	// Seed a portal so we isolate validation errors.
	workspaceID := h.Resources().UserWorkspace.ID
	// The keyspace is created through CreateApi so it has an owning api. One
	// subtest below is a positive case, and the mint-time ceiling needs a real
	// keyspace with an api to express its api-scoped checks against.
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspaceID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	portalID := insertKeyspacePortal(t, h, workspaceID, "test-portal", api.KeyAuthID.String)

	// Granted so validation failures are isolated from authorization: a 400 case
	// must fail on the request body, not on a missing permission.
	rootKey := h.CreateRootKey(workspaceID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
	)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	validScopes := []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"}

	t.Run("missing externalId", func(t *testing.T) {
		req := handler.Request{
			Portal: "test-portal",
			Scopes: validScopes,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty externalId", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "",
			Scopes:     validScopes,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("missing slug", func(t *testing.T) {
		req := handler.Request{
			ExternalId: "user_123",
			Scopes:     validScopes,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty slug", func(t *testing.T) {
		req := handler.Request{
			Portal:     "",
			ExternalId: "user_123",
			Scopes:     validScopes,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("missing permissions", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_123",
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty permissions array", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_123",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("rejects a slug under 3 characters", func(t *testing.T) {
		// Owned by the schema's minLength.
		req := handler.Request{
			Portal:     "ab",
			ExternalId: "user_123",
			Scopes:     validScopes,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("accepts an id-shaped identifier", func(t *testing.T) {
		req := handler.Request{
			Portal:     portalID,
			ExternalId: "user_123",
			Scopes:     validScopes,
		}
		res := testutil.CallRoute[handler.Request, openapi.V2PortalCreateSessionResponseBody](h, route, headers, req)
		require.Equal(t, 200, res.Status)
	})

	t.Run("a well-formed but unknown slug is still a 404", func(t *testing.T) {
		req := handler.Request{
			Portal:     "no-such-portal",
			ExternalId: "user_123",
			Scopes:     validScopes,
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status, "validation must not swallow a genuine not-found")
	})

	// --- Capability vocabulary validation (enforced by the OpenAPI enum) ---

	t.Run("unknown capability rejected", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_123",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:destroy"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("legacy rbac tuple rejected", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_123",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"api.*.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("mixed valid and invalid rejected", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_123",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read", "api.*.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}
