package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	deletekey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_delete_key"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_key"
)

type (
	Request  = deletekey.Request
	Response = deletekey.Response
)

// newHandler builds the portal.deleteKey handler.
func newHandler(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}
}

// setupPortalRoute builds a harness with the portal deleteKey handler registered
// behind the portal middleware stack, plus a freshly created API in the user
// workspace. Every portal session test starts from this identical setup.
func setupPortalRoute(t *testing.T) (*testutil.Harness, *handler.Handler, db.Workspace, db.Api) {
	t.Helper()

	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	workspace := h.Resources().UserWorkspace

	apiName := "Portal DeleteKey Test API"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
		Name:        &apiName,
	})

	return h, route, workspace, api
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

	headers := h.CreatePortalSession(workspace.ID, externalID, []string{
		fmt.Sprintf("api.%s.delete_key", api.ID),
	})

	req := Request{
		KeyId: keyResponse.KeyID,
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
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
	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{
		fmt.Sprintf("api.%s.delete_key", api.ID),
	})

	req := Request{
		KeyId: keyResponse.KeyID,
	}

	res := testutil.CallRoute[Request, openapi.NotFoundErrorResponse](h, route, headers, req)
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

	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{
		fmt.Sprintf("api.%s.delete_key", api.ID),
	})

	req := Request{
		KeyId: keyResponse.KeyID,
	}

	res := testutil.CallRoute[Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, 404, res.Status)
	require.NotNil(t, res.Body)

	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.False(t, key.DeletedAtM.Valid, "key without identity must not be deleted by a portal session")
}
