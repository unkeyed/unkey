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
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

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
// distributed across them for portal scoping tests.
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

// TestProtectedRouteRejectsPortalSession verifies that a portal-session cookie
// cannot authenticate on the protected apis.listKeys route. Portal sessions
// authenticate only on the dedicated portal routes.
func TestProtectedRouteRejectsPortalSession(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}

	// Auth-only stack (no OpenAPI validation) so we assert the auth layer itself
	// rejects the portal session rather than the validator rejecting a missing
	// bearer header.
	authOnly := []zen.Middleware{
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
	h.Register(route, authOnly...)

	setup := setupPortalSessionTest(t, h)

	headers := h.CreatePortalSession(setup.workspace.ID, setup.identity1ExternalID, []string{setup.keySpaceID}, []string{"keys:read"})

	req := handler.Request{
		ApiId: setup.apiID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	// The protected auth stack only reads bearer credentials. A portal_session
	// cookie carries no Authorization header, so auth is treated as missing (400)
	// rather than invalid (401): the protected service is unaware of portal
	// sessions entirely, which is the point of the dedicated portal auth split.
	require.Equal(t, 400, res.Status)
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
