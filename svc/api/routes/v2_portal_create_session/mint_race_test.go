package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
	updateportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// blockedFor is how long a request must stay stuck on the portal lock to count
// as waiting for it.
const blockedFor = 500 * time.Millisecond

// callAsync serves one request on a goroutine and delivers its status code, so
// a test can check whether it's waiting on a lock the test holds.
func callAsync(t *testing.T, h *testutil.Harness, route zen.Route, headers http.Header, body any) <-chan int {
	t.Helper()

	encoded, err := json.Marshal(body)
	require.NoError(t, err)
	status := make(chan int, 1)
	go func() {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewReader(encoded))
		req.Header = headers
		h.Mux().ServeHTTP(rr, req)
		status <- rr.Code
	}()
	return status
}

// A mint that holds the portal lock makes a concurrent disable wait, so the
// disable's revoke runs after the insert and catches the new session.
func TestMintLockMakesDisableRevokeTheNewSession(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	portalID := insertKeyspacePortal(t, h, workspace.ID, "mint-first", api.KeyAuthID.String)

	update := &updateportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(update)
	headers := testutil.RootKeyHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID)))

	mint, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mint.Rollback() })

	locked, err := db.Query.LockPortalForMint(ctx, mint, db.LockPortalForMintParams{ID: portalID, WorkspaceID: workspace.ID})
	require.NoError(t, err)
	require.True(t, locked.Enabled)

	disabled := callAsync(t, h, update, headers, map[string]any{"portal": portalID, "enabled": false})
	select {
	case status := <-disabled:
		t.Fatalf("the disable must wait for the mint's lock, but finished with %d", status)
	case <-time.After(blockedFor):
	}

	now := h.Clock.Now()
	require.NoError(t, db.Query.InsertPortalSession(ctx, mint, db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspace.ID,
		PortalID:              portalID,
		ExternalID:            "user_racing",
		Scopes:                []byte(`{"keyspaceIds":[],"scopes":["keys:read"]}`),
		ExchangeCodeHash:      hash.Sha256(uid.Secure()),
		ExchangeCodeExpiresAt: now.Add(15 * time.Minute).UnixMilli(),
		ReturnUrl:             sql.NullString{Valid: false, String: ""},
		CreatedAt:             now.UnixMilli(),
	}))
	require.NoError(t, mint.Commit())

	require.Equal(t, http.StatusOK, <-disabled)
	require.Equal(t, 0, h.CountLivePortalSessions(t, portalID, ""), "the disable must revoke the session minted before it")
}

// A disable that holds the portal lock makes a concurrent createSession wait,
// and once the disable commits the mint sees it and refuses.
func TestDisableLockMakesMintRefuse(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	portalID := insertKeyspacePortal(t, h, workspace.ID, "disable-first", api.KeyAuthID.String)

	create := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, PortalBaseURL: "https://portal.unkey.com", Clock: h.Clock}
	h.Register(create)
	headers := testutil.RootKeyHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID), fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID), fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID)))

	// Stands in for updatePortal's disable: write the portal row, then revoke.
	disable, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = disable.Rollback() })
	_, err = disable.ExecContext(ctx, "UPDATE portals SET enabled = false WHERE id = ?", portalID)
	require.NoError(t, err)
	_, err = db.Query.RevokePortalSessionsByPortal(ctx, disable, db.RevokePortalSessionsByPortalParams{
		RevokedAt:   sql.NullInt64{Valid: true, Int64: h.Clock.Now().UnixMilli()},
		PortalID:    portalID,
		WorkspaceID: workspace.ID,
	})
	require.NoError(t, err)

	minted := callAsync(t, h, create, headers, handler.Request{
		Portal:     portalID,
		ExternalId: "user_racing",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
	})
	select {
	case status := <-minted:
		t.Fatalf("the mint must wait for the disable's lock, but finished with %d", status)
	case <-time.After(blockedFor):
	}

	require.NoError(t, disable.Commit())

	require.Equal(t, http.StatusForbidden, <-minted, "the mint must see the committed disable")
	require.Equal(t, 0, countPortalSessions(t, h, workspace.ID, "user_racing"), "no session may be written")
}

// A re-point that holds the portal lock makes a concurrent createSession wait.
// The mint already built its grant from the old keyspace, so once the re-point
// commits it must refuse rather than write a session the re-point can't revoke.
func TestRepointLockMakesMintRefuse(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	oldKeyspace := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	newKeyspace := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	portalID := insertKeyspacePortal(t, h, workspace.ID, "repoint-first", oldKeyspace.KeyAuthID.String)

	create := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, PortalBaseURL: "https://portal.unkey.com", Clock: h.Clock}
	h.Register(create)
	headers := testutil.RootKeyHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID), fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID), fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID)))

	// Stands in for updatePortal's re-point: write the portal row, then revoke.
	repoint, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repoint.Rollback() })
	_, err = repoint.ExecContext(ctx, "UPDATE portals SET key_auth_id = ? WHERE id = ?", newKeyspace.KeyAuthID.String, portalID)
	require.NoError(t, err)
	_, err = db.Query.RevokePortalSessionsByPortal(ctx, repoint, db.RevokePortalSessionsByPortalParams{
		RevokedAt:   sql.NullInt64{Valid: true, Int64: h.Clock.Now().UnixMilli()},
		PortalID:    portalID,
		WorkspaceID: workspace.ID,
	})
	require.NoError(t, err)

	minted := callAsync(t, h, create, headers, handler.Request{
		Portal:     portalID,
		ExternalId: "user_repoint",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
	})
	select {
	case status := <-minted:
		t.Fatalf("the mint must wait for the re-point's lock, but finished with %d", status)
	case <-time.After(blockedFor):
	}

	require.NoError(t, repoint.Commit())

	require.Equal(t, http.StatusConflict, <-minted, "the mint must refuse a grant built from the old keyspace")
	require.Equal(t, 0, countPortalSessions(t, h, workspace.ID, "user_repoint"), "no session may be written")
}

// A mint that holds the portal lock makes a concurrent re-point wait, so the
// re-point's revoke runs after the insert and catches the old-scope session.
func TestMintLockMakesRepointRevokeTheNewSession(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	oldKeyspace := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	newKeyspace := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	portalID := insertKeyspacePortal(t, h, workspace.ID, "mint-before-repoint", oldKeyspace.KeyAuthID.String)

	update := &updateportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(update)
	// Re-pointing also needs read access to the new keyspace.
	headers := testutil.RootKeyHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID), fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID)))

	mint, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mint.Rollback() })
	_, err = db.Query.LockPortalForMint(ctx, mint, db.LockPortalForMintParams{ID: portalID, WorkspaceID: workspace.ID})
	require.NoError(t, err)

	repointed := callAsync(t, h, update, headers, map[string]any{"portal": portalID, "keyspaceId": newKeyspace.KeyAuthID.String})
	select {
	case status := <-repointed:
		t.Fatalf("the re-point must wait for the mint's lock, but finished with %d", status)
	case <-time.After(blockedFor):
	}

	now := h.Clock.Now()
	require.NoError(t, db.Query.InsertPortalSession(ctx, mint, db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspace.ID,
		PortalID:              portalID,
		ExternalID:            "user_repoint",
		Scopes:                []byte(fmt.Sprintf(`{"keyspaceIds":[%q],"scopes":["keys:read"]}`, oldKeyspace.KeyAuthID.String)),
		ExchangeCodeHash:      hash.Sha256(uid.Secure()),
		ExchangeCodeExpiresAt: now.Add(15 * time.Minute).UnixMilli(),
		ReturnUrl:             sql.NullString{Valid: false, String: ""},
		CreatedAt:             now.UnixMilli(),
	}))
	require.NoError(t, mint.Commit())

	require.Equal(t, http.StatusOK, <-repointed)
	require.Equal(t, 0, h.CountLivePortalSessions(t, portalID, ""), "the re-point must revoke the session minted before it")
}
