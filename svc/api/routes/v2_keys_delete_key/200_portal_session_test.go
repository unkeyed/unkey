package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_delete_key"
)

// portalMiddleware returns a middleware stack that authenticates requests
// (including portal session cookies) but skips OpenAPI spec validation. The
// OpenAPI spec only declares rootKey security, so cookie-authenticated portal
// requests would be rejected by the validator.
func portalMiddleware(h *testutil.Harness) []zen.Middleware {
	return []zen.Middleware{
		zen.WithObservability(),
		zen.WithLogging(),
		middleware.WithErrorHandling(),
		middleware.WithAuthentication(middleware.AuthenticationConfig{
			Auth:       h.Auth,
			Database:   h.DB,
			QuotaCache: h.Caches.WorkspaceQuota,
			Ratelimit:  h.Ratelimit,
		}),
	}
}

// setupPortalRoute builds a harness with the delete-key handler registered
// behind the portal middleware stack, plus a freshly created API in the user
// workspace. Every portal session test starts from this identical setup.
func setupPortalRoute(t *testing.T) (*testutil.Harness, *handler.Handler, db.Workspace, db.Api) {
	t.Helper()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}
	h.Register(route, portalMiddleware(h)...)

	workspace := h.Resources().UserWorkspace

	apiName := "Portal DeleteKey Test API"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
		Name:        &apiName,
	})

	return h, route, workspace, api
}

// createPortalSession inserts a portal session row and returns a cookie header
// suitable for use in CallRoute.
func createPortalSession(
	t *testing.T,
	h *testutil.Harness,
	workspaceID string,
	externalID string,
	permissions []string,
) http.Header {
	t.Helper()
	ctx := context.Background()

	sessionID := uid.New(uid.PortalSessionPrefix)

	permsJSON, err := json.Marshal(permissions)
	require.NoError(t, err)

	err = db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:             sessionID,
		WorkspaceID:    workspaceID,
		PortalConfigID: uid.New(uid.PortalConfigPrefix),
		ExternalID:     externalID,
		Permissions:    permsJSON,
		Preview:        false,
		ExpiresAt:      time.Now().Add(24 * time.Hour).UnixMilli(),
		CreatedAt:      time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	return http.Header{
		"Content-Type": {"application/json"},
		"Cookie":       {fmt.Sprintf("portal_session=%s", sessionID)},
	}
}

// TestPortalSessionDeleteOwnKey verifies a portal session can delete a key that
// belongs to its own externalId identity.
func TestPortalSessionDeleteOwnKey(t *testing.T) {
	ctx := context.Background()
	h, route, workspace, api := setupPortalRoute(t)

	externalID := "portal_user_A"
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalID,
	})

	keyName := "portal-owned-key"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
		IdentityID:  ptr.P(identity.ID),
	})

	headers := createPortalSession(t, h, workspace.ID, externalID, []string{
		fmt.Sprintf("api.%s.delete_key", api.ID),
	})

	req := handler.Request{
		KeyId: keyResponse.KeyID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)

	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.True(t, key.DeletedAtM.Valid, "key should be soft deleted")
}

// TestPortalSessionCannotDeleteOtherIdentityKey verifies a portal session
// cannot delete a key belonging to a different externalId. The handler returns
// 404 to avoid leaking the existence of keys the session does not own.
func TestPortalSessionCannotDeleteOtherIdentityKey(t *testing.T) {
	ctx := context.Background()
	h, route, workspace, api := setupPortalRoute(t)

	// Key belongs to user B.
	otherIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "portal_user_B",
	})

	keyName := "user-b-key"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
		IdentityID:  ptr.P(otherIdentity.ID),
	})

	// Session is authenticated as user A but has the permission to delete keys.
	headers := createPortalSession(t, h, workspace.ID, "portal_user_A", []string{
		fmt.Sprintf("api.%s.delete_key", api.ID),
	})

	req := handler.Request{
		KeyId: keyResponse.KeyID,
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, 404, res.Status)
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Detail, "The specified key was not found")

	// The key must still exist and not be deleted.
	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.False(t, key.DeletedAtM.Valid, "key belonging to another identity must not be deleted")
}

// TestPortalSessionCannotDeleteKeyWithoutIdentity verifies a portal session
// cannot delete a key that has no identity at all. Such a key can never belong
// to the session's externalId, so it returns 404.
func TestPortalSessionCannotDeleteKeyWithoutIdentity(t *testing.T) {
	ctx := context.Background()
	h, route, workspace, api := setupPortalRoute(t)

	keyName := "no-identity-key"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
	})

	headers := createPortalSession(t, h, workspace.ID, "portal_user_A", []string{
		fmt.Sprintf("api.%s.delete_key", api.ID),
	})

	req := handler.Request{
		KeyId: keyResponse.KeyID,
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, 404, res.Status)
	require.NotNil(t, res.Body)

	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.False(t, key.DeletedAtM.Valid, "key without identity must not be deleted by a portal session")
}
