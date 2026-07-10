package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	rerollkey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_reroll_key"
)

type (
	Request  = rerollkey.Request
	Response = rerollkey.Response
)

// newHandler builds the portal.rerollKey handler backed by a configured
// keys.rerollKey handler.
func newHandler(h *testutil.Harness) *handler.Handler {
	return handler.New(&rerollkey.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	})
}

// TestPortalSessionRerollOwnKey verifies a portal session can reroll a key that
// belongs to its own externalId identity, and that a fresh secret is returned.
func TestPortalSessionRerollOwnKey(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

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

	headers := h.CreatePortalSession(workspace.ID, externalID, []string{api.KeyAuthID.String}, []string{"keys:reroll"})

	req := Request{
		KeyId: key.KeyID,
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)

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

func TestPortalSessionRequiresRerollCapability(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "portal_user_A",
	})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  ptr.P(identity.ID),
	})

	headers := h.CreatePortalSession(
		workspace.ID,
		identity.ExternalID,
		[]string{api.KeyAuthID.String},
		[]string{"keys:create"},
	)
	owned := testutil.CallRoute[Request, openapi.ForbiddenErrorResponse](h, route, headers, Request{KeyId: key.KeyID})
	missingKeyID := uid.New(uid.KeyPrefix)
	missing := testutil.CallRoute[Request, openapi.ForbiddenErrorResponse](h, route, headers, Request{KeyId: missingKeyID})

	require.Equal(t, http.StatusForbidden, owned.Status, "keys:create must not authorize portal.rerollKey")
	require.Equal(t, owned.Body.Error, missing.Body.Error, "authorization must run before key lookup")
	require.NotContains(t, owned.RawBody, key.KeyID)
	require.NotContains(t, missing.RawBody, missingKeyID)
}

func TestPortalSessionCannotRerollKeyOutsideSessionKeyspaces(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	workspace := h.Resources().UserWorkspace
	keyApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	sessionApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "portal_user_A",
	})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  keyApi.KeyAuthID.String,
		IdentityID:  ptr.P(identity.ID),
	})

	headers := h.CreatePortalSession(
		workspace.ID,
		identity.ExternalID,
		[]string{sessionApi.KeyAuthID.String},
		[]string{"keys:reroll"},
	)
	res := testutil.CallRoute[Request, Response](h, route, headers, Request{KeyId: key.KeyID})

	require.Equal(t, 404, res.Status, "keys outside the session keyspaces must be hidden")
}

// TestPortalSessionCannotRerollOtherIdentityKey verifies a portal session
// cannot reroll a key belonging to a different externalId and receives a 404
// so the key's existence is not leaked.
func TestPortalSessionCannotRerollOtherIdentityKey(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

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

	// Session belongs to user A and can reroll its own keys.
	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{api.KeyAuthID.String}, []string{"keys:reroll"})

	req := Request{
		KeyId: otherKey.KeyID,
	}

	res := testutil.CallRoute[Request, openapi.NotFoundErrorResponse](h, route, headers, req)

	require.Equal(t, 404, res.Status, "rerolling another identity's key should return 404")
	require.Equal(t, "The specified key was not found.", res.Body.Error.Detail)
	require.NotContains(t, res.RawBody, otherKey.KeyID)
}

// TestPortalSessionCannotRerollKeyWithoutIdentity verifies a portal session
// cannot reroll a key that has no identity assigned (returns 404).
func TestPortalSessionCannotRerollKeyWithoutIdentity(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyWithoutIdentity := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{api.KeyAuthID.String}, []string{"keys:reroll"})

	req := Request{
		KeyId: keyWithoutIdentity.KeyID,
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)

	require.Equal(t, 404, res.Status, "rerolling an unowned key should return 404")
}
