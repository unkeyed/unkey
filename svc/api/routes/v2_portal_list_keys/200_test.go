package handler_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	listkeys "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_keys"
)

type (
	Request  = listkeys.Request
	Response = listkeys.Response
)

// newHandler builds the portal.listKeys handler backed by a configured
// apis.listKeys handler.
func newHandler(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		Handler: &listkeys.Handler{
			DB:       h.DB,
			Vault:    h.Vault,
			ApiCache: h.Caches.LiveApiByID,
		},
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

func TestPortalSessionScopesToOwnExternalID(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	setup := setupPortalSessionTest(t, h)

	// Portal session for user A with read permissions
	headers := h.CreatePortalSession(setup.workspace.ID, setup.identity1ExternalID, []string{setup.keySpaceID}, []string{"keys:read"})

	req := Request{
		ApiId: setup.apiID,
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)

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

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	setup := setupPortalSessionTest(t, h)

	// Portal session for user A
	headers := h.CreatePortalSession(setup.workspace.ID, setup.identity1ExternalID, []string{setup.keySpaceID}, []string{"keys:read"})

	// Attempt to supply user B's externalId — should be overridden
	req := Request{
		ApiId:      setup.apiID,
		ExternalId: &setup.identity2ExternalID,
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)

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

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	setup := setupPortalSessionTest(t, h)

	// Portal session for a user that has no identity record
	headers := h.CreatePortalSession(setup.workspace.ID, "non_existent_user", []string{setup.keySpaceID}, []string{"keys:read"})

	req := Request{
		ApiId: setup.apiID,
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)

	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.Data)
	require.Len(t, res.Body.Data, 0)
	require.False(t, res.Body.Pagination.HasMore)
}
