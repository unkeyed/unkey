package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	portalID := uid.New(uid.PortalPrefix)
	now := time.Now().UnixMilli()

	// A keyspace-mapped portal: the session is scoped to this keyspace,
	// derived from the config rather than the request.
	keySpaceID := uid.New(uid.KeySpacePrefix)
	require.NoError(t, db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID,
		WorkspaceID:   workspaceID,
		CreatedAtM:    now,
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	}))

	err := db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          portalID,
		WorkspaceID: workspaceID,
		Slug:        "test-portal",
		KeyspaceID:  sql.NullString{Valid: true, String: keySpaceID},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("basic session creation", func(t *testing.T) {
		req := handler.Request{
			Portal:      "test-portal",
			ExternalId:  "user_123",
			Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.SessionId)
		require.NotEmpty(t, res.Body.Data.Url)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// URL must contain the exchange code and the portal base URL.
		require.Contains(t, res.Body.Data.Url, "portal.unkey.com")
		require.True(t, strings.HasPrefix(res.Body.Data.Url, "https://"))
		portalURL, err := url.Parse(res.Body.Data.Url)
		require.NoError(t, err)
		require.NotEmpty(t, portalURL.Query().Get("code"))
		require.Empty(t, portalURL.Fragment)
	})

	t.Run("portal ID", func(t *testing.T) {
		req := handler.Request{
			Portal:      portalID,
			ExternalId:  "user_by_portal_id",
			Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotEmpty(t, res.Body.Data.SessionId)
		require.NotEmpty(t, exchangeCodeFromPortalURL(t, res.Body.Data.Url))
	})

	t.Run("persists hashed exchange code", func(t *testing.T) {
		req := handler.Request{
			Portal:      "test-portal",
			ExternalId:  "user_789",
			Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.SessionId)
		exchangeCode := exchangeCodeFromPortalURL(t, res.Body.Data.Url)

		// Verify the pending session was persisted with a hashed exchange code.
		session, err := db.Query.FindValidPortalSessionByExchangeCode(ctx, h.DB.RO(), db.FindValidPortalSessionByExchangeCodeParams{
			ExchangeCodeHash: sql.NullString{String: hash.Sha256(exchangeCode), Valid: true},
			Now:              time.Now().UnixMilli(),
		})
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.SessionId, session.ID)
		require.Equal(t, "user_789", session.ExternalID)
		require.Equal(t, hash.Sha256(exchangeCode), session.ExchangeCodeHash.String)
		require.NotEqual(t, exchangeCode, session.ExchangeCodeHash.String)
		require.False(t, session.AccessTokenHash.Valid)
	})

	t.Run("multiple sessions for same externalId", func(t *testing.T) {
		req := handler.Request{
			Portal:      "test-portal",
			ExternalId:  "user_multi",
			Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)

		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)

		// Each call must produce unique session IDs and exchange codes.
		require.NotEqual(t, res1.Body.Data.SessionId, res2.Body.Data.SessionId)
		require.NotEqual(t, exchangeCodeFromPortalURL(t, res1.Body.Data.Url), exchangeCodeFromPortalURL(t, res2.Body.Data.Url))
	})
}

func exchangeCodeFromPortalURL(t *testing.T, rawURL string) string {
	t.Helper()

	portalURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	exchangeCode := portalURL.Query().Get("code")
	require.NotEmpty(t, exchangeCode)
	return exchangeCode
}
