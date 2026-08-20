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

// exchangeCodeFromURL extracts the single-use code the portal URL carries.
func exchangeCodeFromURL(t *testing.T, portalURL string) string {
	t.Helper()

	parsed, err := url.Parse(portalURL)
	require.NoError(t, err)

	code := parsed.Query().Get("code")
	require.NotEmpty(t, code)
	return code
}

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

	// A keyspace-mapped portal: the session is scoped to this keyspace, derived
	// from the portal rather than the request.
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
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
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
			Portal:     "test-portal",
			ExternalId: "user_123",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.Id)
		require.NotEmpty(t, res.Body.Data.Url)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// The response returns the non-secret handle; the credential lives only
		// in the URL, so the id must not appear there.
		require.True(t, strings.HasPrefix(res.Body.Data.Id, "ps_"))
		require.Contains(t, res.Body.Data.Url, "portal.unkey.com")
		require.NotContains(t, res.Body.Data.Url, res.Body.Data.Id)
		require.True(t, strings.HasPrefix(res.Body.Data.Url, "https://"))
	})

	t.Run("resolves the portal by id as well as slug", func(t *testing.T) {
		req := handler.Request{
			Portal:     portalID,
			ExternalId: "user_by_id",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotEmpty(t, res.Body.Data.Id)
	})

	t.Run("with preview", func(t *testing.T) {
		preview := true
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_789",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
			Preview:    &preview,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.Id)

		// The session persists in the pending state: the code's hash is stored,
		// no access token has been issued.
		code := exchangeCodeFromURL(t, res.Body.Data.Url)
		session, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), hash.Sha256(code))
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.Id, session.ID)
		require.Equal(t, "user_789", session.ExternalID)
		require.Equal(t, portalID, session.PortalID)
		require.True(t, session.Preview)
		require.False(t, session.AccessTokenHash.Valid)
	})

	t.Run("stores the exchange code only as a hash", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_hashed",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		code := exchangeCodeFromURL(t, res.Body.Data.Url)

		// The plaintext code must not be queryable: only its hash resolves.
		_, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), code)
		require.True(t, db.IsNotFound(err), "plaintext exchange code must not be stored")

		session, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), hash.Sha256(code))
		require.NoError(t, err)
		require.NotEqual(t, code, session.ExchangeCodeHash)
	})

	t.Run("multiple sessions for same externalId", func(t *testing.T) {
		req := handler.Request{
			Portal:     "test-portal",
			ExternalId: "user_multi",
			Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)

		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)

		// Each call must produce a unique session and a unique code.
		require.NotEqual(t, res1.Body.Data.Id, res2.Body.Data.Id)
		require.NotEqual(t, res1.Body.Data.Url, res2.Body.Data.Url)
	})
}
