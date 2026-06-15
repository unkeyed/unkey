package handler_test

import (
	"context"
	"database/sql"
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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

// portalMiddleware returns a middleware stack that authenticates requests
// (including portal session cookies) but skips OpenAPI spec validation.
// The OpenAPI spec only declares rootKey security, so cookie-authenticated
// portal requests would be rejected by the validator.
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

// portalSessionSetup holds all objects created for a portal session test scenario.
type portalSessionSetup struct {
	apiID      string
	keySpaceID string
	workspace  db.Workspace

	identity1ID         string
	identity1ExternalID string
	identity2ID         string
	identity2ExternalID string

	key1ID string // belongs to identity1
	key2ID string // belongs to identity1
	key3ID string // belongs to identity2
	key4ID string // no identity
}

// setupPortalSessionTest creates a workspace, API, two identities, and keys
// distributed across them for portal session testing.
func setupPortalSessionTest(t *testing.T, h *testutil.Harness) portalSessionSetup {
	t.Helper()
	ctx := context.Background()

	workspace := h.Resources().UserWorkspace

	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	apiID := uid.New("api")
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "Portal Test API",
		WorkspaceID: workspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	identity1ExternalID := "portal_user_A"
	identity1 := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  identity1ExternalID,
	})

	identity2ExternalID := "portal_user_B"
	identity2 := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  identity2ExternalID,
	})

	key1 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Key 1 - User A"),
		IdentityID:  ptr.P(identity1.ID),
	})

	key2 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Key 2 - User A"),
		IdentityID:  ptr.P(identity1.ID),
	})

	key3 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Key 3 - User B"),
		IdentityID:  ptr.P(identity2.ID),
	})

	key4 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keySpaceID,
		Name:        ptr.P("Key 4 - No identity"),
	})

	return portalSessionSetup{
		apiID:               apiID,
		keySpaceID:          keySpaceID,
		workspace:           workspace,
		identity1ID:         identity1.ID,
		identity1ExternalID: identity1ExternalID,
		identity2ID:         identity2.ID,
		identity2ExternalID: identity2ExternalID,
		key1ID:              key1.KeyID,
		key2ID:              key2.KeyID,
		key3ID:              key3.KeyID,
		key4ID:              key4.KeyID,
	}
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

func TestPortalSessionScopesToOwnExternalID(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}
	h.Register(route, portalMiddleware(h)...)

	setup := setupPortalSessionTest(t, h)

	// Portal session for user A with read permissions
	headers := createPortalSession(t, h, setup.workspace.ID, setup.identity1ExternalID, []string{
		fmt.Sprintf("api.%s.read_key", setup.apiID),
		fmt.Sprintf("api.%s.read_api", setup.apiID),
	})

	req := handler.Request{
		ApiId: setup.apiID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.Data)
	// Only keys belonging to identity1 (user A) should be returned
	require.Len(t, res.Body.Data, 2)

	returnedIDs := map[string]bool{}
	for _, key := range res.Body.Data {
		returnedIDs[key.KeyId] = true
		require.NotNil(t, key.Identity)
		require.Equal(t, setup.identity1ExternalID, key.Identity.ExternalId)
	}
	require.True(t, returnedIDs[setup.key1ID], "key1 should be in results")
	require.True(t, returnedIDs[setup.key2ID], "key2 should be in results")
}

func TestPortalSessionOverridesSuppliedExternalID(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}
	h.Register(route, portalMiddleware(h)...)

	setup := setupPortalSessionTest(t, h)

	// Portal session for user A
	headers := createPortalSession(t, h, setup.workspace.ID, setup.identity1ExternalID, []string{
		fmt.Sprintf("api.%s.read_key", setup.apiID),
		fmt.Sprintf("api.%s.read_api", setup.apiID),
	})

	// Attempt to supply user B's externalId — should be overridden
	req := handler.Request{
		ApiId:      setup.apiID,
		ExternalId: &setup.identity2ExternalID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.Data)
	// Should still only see user A's keys, not user B's
	require.Len(t, res.Body.Data, 2)

	for _, key := range res.Body.Data {
		require.NotNil(t, key.Identity)
		require.Equal(t, setup.identity1ExternalID, key.Identity.ExternalId)
	}
}

func TestPortalSessionNonExistentIdentityReturnsEmpty(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}
	h.Register(route, portalMiddleware(h)...)

	setup := setupPortalSessionTest(t, h)

	// Portal session for a user that has no identity record
	headers := createPortalSession(t, h, setup.workspace.ID, "non_existent_user", []string{
		fmt.Sprintf("api.%s.read_key", setup.apiID),
		fmt.Sprintf("api.%s.read_api", setup.apiID),
	})

	req := handler.Request{
		ApiId: setup.apiID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.Data)
	require.Len(t, res.Body.Data, 0)
	require.False(t, res.Body.Pagination.HasMore)
}

func TestRootKeyUnaffectedByPortalScoping(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}
	h.Register(route)

	setup := setupPortalSessionTest(t, h)

	rootKey := h.CreateRootKey(setup.workspace.ID, "api.*.read_key", "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("root key lists all keys without externalId filter", func(t *testing.T) {
		req := handler.Request{
			ApiId: setup.apiID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body.Data)
		// Should see all 4 keys (identity1 x2, identity2 x1, no-identity x1)
		require.Len(t, res.Body.Data, 4)
	})

	t.Run("root key filters by externalId normally", func(t *testing.T) {
		req := handler.Request{
			ApiId:      setup.apiID,
			ExternalId: &setup.identity2ExternalID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, setup.key3ID, res.Body.Data[0].KeyId)
		require.NotNil(t, res.Body.Data[0].Identity)
		require.Equal(t, setup.identity2ExternalID, res.Body.Data[0].Identity.ExternalId)
	})
}
