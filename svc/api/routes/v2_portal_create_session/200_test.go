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
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		Keys:          h.Keys,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

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

	rootKey := h.CreateRootKey(workspaceID, rbac.Tuple{
		ResourceType: rbac.Portal,
		ResourceID:   "*",
		Action:       rbac.CreatePortalSession,
	}.String(), "api.*.read_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("basic session creation", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_123",
			Permissions: []string{"api.*.read_key"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.SessionId)
		require.NotEmpty(t, res.Body.Data.Url)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// URL must contain the session ID and the portal base URL.
		require.Contains(t, res.Body.Data.Url, "portal.unkey.com")
		require.Contains(t, res.Body.Data.Url, res.Body.Data.SessionId)
		require.True(t, strings.HasPrefix(res.Body.Data.Url, "https://"))
	})

	t.Run("with preview", func(t *testing.T) {
		preview := true
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_789",
			Permissions: []string{"api.*.read_key"},
			Preview:     &preview,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.SessionId)

		// Verify the token was persisted with correct fields.
		token, err := db.Query.FindValidPortalSessionToken(ctx, h.DB.RO(), db.FindValidPortalSessionTokenParams{
			ID:  res.Body.Data.SessionId,
			Now: time.Now().UnixMilli(),
		})
		require.NoError(t, err)
		require.Equal(t, "user_789", token.ExternalID)
		require.True(t, token.Preview)
	})

	t.Run("multiple sessions for same externalId", func(t *testing.T) {
		req := handler.Request{
			Slug:        "test-portal",
			ExternalId:  "user_multi",
			Permissions: []string{"api.*.read_key"},
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)

		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)

		// Each call must produce a unique session ID.
		require.NotEqual(t, res1.Body.Data.SessionId, res2.Body.Data.SessionId)
	})
}
