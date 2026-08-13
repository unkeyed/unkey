package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_keys"
)

type (
	Request  = openapi.V2PortalListKeysRequestBody
	Response = openapi.V2PortalListKeysResponseBody
)

// newHandler builds the portal.listKeys handler.
func newHandler(h *testutil.Harness) *handler.Handler {
	return handler.New(h.DB)
}

// portalSessionSetup holds all objects created for a portal session test scenario.
type portalSessionSetup struct {
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

// setupPortalSessionTest creates a workspace, keyspace, two identities, and keys
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

	res := testutil.CallRoute[Request, Response](h, route, headers, Request{})

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

// TestPortalSessionListKeysRequiresKeysRead guarantees that another key
// capability cannot expose identity or key data through the listing endpoint.
func TestPortalSessionListKeysRequiresKeysRead(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	setup := setupPortalSessionTest(t, h)
	headers := h.CreatePortalSession(
		setup.workspace.ID,
		setup.identity1ExternalID,
		[]string{setup.keySpaceID},
		[]string{"keys:reroll"},
	)

	res := testutil.CallRoute[Request, openapi.ForbiddenErrorResponse](h, route, headers, Request{})

	require.Equal(t, 403, res.Status, "keys:reroll must not authorize portal.listKeys")
	require.NotContains(t, res.RawBody, setup.workspace.ID)
	require.NotContains(t, res.RawBody, setup.identity1ExternalID)
	require.NotContains(t, res.RawBody, setup.key1ID)
}

func TestPortalSessionUnionsConfiguredKeyspaces(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	setup := setupPortalSessionTest(t, h)
	ctx := context.Background()

	// A second keyspace with another key for user A. A session scoped to both
	// keyspaces must return user A's keys from each of them.
	keySpaceID2 := uid.New(uid.KeySpacePrefix)
	require.NoError(t, db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID2,
		WorkspaceID:   setup.workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	}))
	key5 := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: setup.workspace.ID,
		KeySpaceID:  keySpaceID2,
		Name:        ptr.P("Key 5 - User A, keyspace 2"),
		IdentityID:  ptr.P(setup.identity1ID),
	})

	headers := h.CreatePortalSession(
		setup.workspace.ID,
		setup.identity1ExternalID,
		[]string{setup.keySpaceID, keySpaceID2},
		[]string{"keys:read"},
	)

	res := testutil.CallRoute[Request, Response](h, route, headers, Request{})

	require.Equal(t, 200, res.Status)
	require.Len(t, res.Body.Data, 3, "user A's keys from both keyspaces")

	returnedIDs := map[string]bool{}
	for _, key := range res.Body.Data {
		returnedIDs[key.KeyId] = true
	}
	require.True(t, returnedIDs[setup.key1ID])
	require.True(t, returnedIDs[setup.key2ID])
	require.True(t, returnedIDs[key5.KeyID], "key from the second keyspace should be included")
}

func TestPortalSessionNonExistentIdentityReturnsEmpty(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	setup := setupPortalSessionTest(t, h)

	// Portal session for a user that has no identity record
	headers := h.CreatePortalSession(setup.workspace.ID, "non_existent_user", []string{setup.keySpaceID}, []string{"keys:read"})

	res := testutil.CallRoute[Request, Response](h, route, headers, Request{})

	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body.Data)
	require.Len(t, res.Body.Data, 0)
	require.False(t, res.Body.Pagination.HasMore)
}

// TestPortalSessionIsRejectedAfterCachedSessionExpires guarantees that the
// authentication cache cannot extend a portal session beyond its expiry.
func TestPortalSessionIsRejectedAfterCachedSessionExpires(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	setup := setupPortalSessionTest(t, h)
	ctx := context.Background()
	sessionID := uid.New(uid.PortalSessionPrefix)
	exchangeCode := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
	accessToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()

	scopes, err := json.Marshal(struct {
		KeyspaceIDs []string `json:"keyspaceIds"`
		Scopes      []string `json:"scopes"`
	}{
		KeyspaceIDs: []string{setup.keySpaceID},
		Scopes:      []string{"keys:read"},
	})
	require.NoError(t, err)

	now := h.Clock.Now()
	require.NoError(t, db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    sessionID,
		WorkspaceID:           setup.workspace.ID,
		PortalID:              uid.New(uid.PortalConfigPrefix),
		ExternalID:            setup.identity1ExternalID,
		Scopes:                scopes,
		ExchangeCodeHash:      hash.Sha256(exchangeCode),
		ExchangeCodeExpiresAt: now.Add(time.Minute).UnixMilli(),
		CreatedAt:             now.UnixMilli(),
	}))

	// An access token that falls due one minute from now.
	redeemed, err := db.Query.ExchangePortalSessionCode(ctx, h.DB.RW(), db.ExchangePortalSessionCodeParams{
		AccessTokenHash:      sql.NullString{String: hash.Sha256(accessToken), Valid: true},
		AccessTokenCreatedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		AccessTokenExpiresAt: sql.NullInt64{Int64: now.Add(time.Minute).UnixMilli(), Valid: true},
		ExchangeCodeHash:     hash.Sha256(exchangeCode),
		Now:                  now.UnixMilli(),
	})
	require.NoError(t, err)
	affected, err := redeemed.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	headers := http.Header{
		"Content-Type": {"application/json"},
		"Cookie":       {"portal_session=" + accessToken},
	}
	warm := testutil.CallRoute[Request, Response](h, route, headers, Request{})
	require.Equal(t, http.StatusOK, warm.Status)

	h.Clock.Tick(time.Minute)
	expired := testutil.CallRoute[Request, openapi.UnauthorizedErrorResponse](h, route, headers, Request{})
	require.Equal(t, http.StatusUnauthorized, expired.Status)
	require.Equal(t, "The portal session is invalid or has expired.", expired.Body.Error.Detail)
	require.NotContains(t, expired.RawBody, sessionID)
	require.NotContains(t, expired.RawBody, accessToken)
}
