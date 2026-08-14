package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	// Seed a portal so we isolate validation errors.
	workspaceID := h.Resources().UserWorkspace.ID
	portalID := uid.New(uid.PortalPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          portalID,
		WorkspaceID: workspaceID,
		Slug:        "test-portal",
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(workspaceID)

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

	// --- Identifier shape validation (id-shaped or slug-shaped, nothing between) ---
	//
	// The shared ResourceIdentifier schema has to admit both shapes, so it alone
	// would let these through to the lookup and answer 404. They are malformed
	// input, not a missing portal, and must say so.

	for _, tt := range []struct {
		name   string
		portal string
	}{
		{name: "uppercase slug", portal: "Test-Portal"},
		{name: "consecutive hyphens", portal: "test--portal"},
		{name: "leading hyphen", portal: "-test-portal"},
		{name: "trailing hyphen", portal: "test-portal-"},
		{name: "slug over 64 characters", portal: strings.Repeat("a", 65)},
	} {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			req := handler.Request{
				Portal:     tt.portal,
				ExternalId: "user_123",
				Scopes:     validScopes,
			}
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
			require.Equal(t, 400, res.Status, "malformed identifier must be an input error, not a 404")
			require.NotNil(t, res.Body)
			require.Contains(t, res.Body.Error.Detail, "slug",
				"the message must name the rule that was broken")
		})
	}

	t.Run("rejects a slug under 3 characters", func(t *testing.T) {
		// Owned by the schema's minLength, which runs before the handler, so
		// this one carries the generic schema message rather than the slug rule.
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
		// Same lookup path, but the id branch skips the slug rules entirely.
		// A 400 here would mean the split rejected a legitimate id.
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
