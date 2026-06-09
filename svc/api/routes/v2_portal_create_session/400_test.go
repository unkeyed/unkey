package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
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

	// Seed a portal config so we isolate validation errors.
	workspaceID := h.Resources().UserWorkspace.ID
	portalConfigID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          portalConfigID,
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

	t.Run("missing externalId", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			Permissions: []string{"api.*.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty externalId", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "",
			Permissions: []string{"api.*.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("missing permissions", func(t *testing.T) {
		req := handler.Request{
			Slug:       "test-portal",
			ExternalId: "user_123",
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty permissions array", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("missing slug", func(t *testing.T) {
		req := handler.Request{
			ExternalId:  "user_123",
			Permissions: []string{"api.*.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty slug", func(t *testing.T) {
		req := handler.Request{
			Slug:        "",
			ExternalId:  "user_123",
			Permissions: []string{"api.*.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	// --- Permission format validation tests ---

	t.Run("old format rejected", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{"keys:read"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("two segments rejected", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{"api.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("four segments rejected", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{"api.*.read_key.extra"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty middle segment rejected", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{"api..read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty first segment rejected", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{".*.read_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty last segment rejected", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{"api.*."},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("mixed valid and invalid rejected", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{"api.*.read_key", "keys:read"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}
