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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
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

// TestPortalSessionRerollOwnKey verifies a portal session can reroll a key that
// belongs to its own externalId identity, and that a fresh secret is returned.
func TestPortalSessionRerollOwnKey(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}
	h.Register(route, portalMiddleware(h)...)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	externalID := "portal_user_A"
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalID,
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identity.ID),
	})

	headers := createPortalSession(t, h, workspace.ID, externalID, []string{
		fmt.Sprintf("api.%s.create_key", api.ID),
	})

	req := handler.Request{
		KeyId: key.KeyID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Data.KeyId)
	require.NotEmpty(t, res.Body.Data.Key, "new key secret should be returned")
	require.NotEqual(t, key.KeyID, res.Body.Data.KeyId, "reroll should produce a new key id")

	// The new key should be owned by the same identity.
	newKey, err := db.Query.FindKeyByID(ctx, h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.True(t, newKey.IdentityID.Valid)
	require.Equal(t, identity.ID, newKey.IdentityID.String)
}

// TestPortalSessionCannotRerollOtherIdentityKey verifies a portal session
// cannot reroll a key belonging to a different externalId and receives a 404
// so the key's existence is not leaked.
func TestPortalSessionCannotRerollOtherIdentityKey(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}
	h.Register(route, portalMiddleware(h)...)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Key owned by user B.
	otherIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "portal_user_B",
	})
	otherKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(otherIdentity.ID),
	})

	// Session belongs to user A but holds create_key permission on the API.
	headers := createPortalSession(t, h, workspace.ID, "portal_user_A", []string{
		fmt.Sprintf("api.%s.create_key", api.ID),
	})

	req := handler.Request{
		KeyId: otherKey.KeyID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	require.Equal(t, 404, res.Status, "rerolling another identity's key should return 404")
}

// TestPortalSessionCannotRerollKeyWithoutIdentity verifies a portal session
// cannot reroll a key that has no identity assigned (returns 404).
func TestPortalSessionCannotRerollKeyWithoutIdentity(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}
	h.Register(route, portalMiddleware(h)...)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyWithoutIdentity := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	headers := createPortalSession(t, h, workspace.ID, "portal_user_A", []string{
		fmt.Sprintf("api.%s.create_key", api.ID),
	})

	req := handler.Request{
		KeyId: keyWithoutIdentity.KeyID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	require.Equal(t, 404, res.Status, "rerolling an unowned key should return 404")
}
