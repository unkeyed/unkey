package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_reroll_key"
)

// TestRerollRootKeyCopiesPermissionsAndRevokesOriginal guarantees rotation keeps
// access while expiration 0 revokes the original even when its cache is warm.
func TestRerollRootKeyCopiesPermissionsAndRevokesOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{permission}})
	sourceSession := rootKeySession(t, source.Key)
	_, err := h.Keys.GetRootKey(t.Context(), sourceSession)
	require.NoError(t, err)
	caller := h.CreateRootKey(workspace.ID,
		"unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#write",
		"unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#delete",
		permission,
	)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue[int64](0)})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), sourceSession)
	require.Error(t, err)
	rerolled, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, res.Body.Data.Key))
	require.NoError(t, err)
	require.Equal(t, []string{permission}, rerolled.Permissions)
}

// TestRerollRootKeyExpirationRequiresDeletePermission guarantees callers cannot
// shorten or revoke an original root key with write access alone.
func TestRerollRootKeyExpirationRequiresDeletePermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#write")
	before := countNewRootKeys(t, h, workspace.ID)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue[int64](0)})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, before, countNewRootKeys(t, h, workspace.ID))
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, source.Key))
	require.NoError(t, err)
}

// TestRerollRootKeyRequiresExplicitNullToKeepOriginal guarantees callers must
// choose whether the original survives, and null does not need delete access.
func TestRerollRootKeyRequiresExplicitNullToKeepOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#write")
	missing := testutil.CallRoute[map[string]any, handler.Response](h, route, headers(caller), map[string]any{"keyId": source.KeyID})
	require.Equal(t, http.StatusBadRequest, missing.Status, "%s", missing.RawBody)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, source.Key))
	require.NoError(t, err)
}

// TestRerollRootKeyRejectsPermissionEscalationWithoutWrites guarantees a caller
// cannot receive a secret for permissions it does not already hold.
func TestRerollRootKeyRejectsPermissionEscalationWithoutWrites(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{"unkey:v1:" + workspace.ID + ":**#*"}})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	before := countNewRootKeys(t, h, workspace.ID)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, before, countNewRootKeys(t, h, workspace.ID))
}

// TestRerollRootKeyOverlapCannotExtendOriginal guarantees a grace period never
// extends the original key's existing expiration.
func TestRerollRootKeyOverlapCannotExtendOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	expires := h.Clock.Now().Add(time.Hour).Truncate(time.Millisecond)
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Expires: &expires})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", "unkey:v1:"+workspace.ID+":rootKeys/*#delete")
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue((24 * time.Hour).Milliseconds())})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	original, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), source.KeyID)
	require.NoError(t, err)
	require.Equal(t, expires.UnixMilli(), original.Expires.Int64)
	rerolled, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, expires.UnixMilli(), rerolled.Expires.Int64)
}

// TestRerollRootKeyPreservesSourceExpiration guarantees the replacement
// inherits the source key's expiration even when the caller expires earlier.
func TestRerollRootKeyPreservesSourceExpiration(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	sourceExpires := h.Clock.Now().Add(2 * time.Hour).Truncate(time.Millisecond)
	callerExpires := h.Clock.Now().Add(time.Hour).Truncate(time.Millisecond)
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Expires: &sourceExpires})
	caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Expires:     &callerExpires,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/" + source.KeyID + "#write"},
	})

	res := call(h, route, caller.Key, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	rerolled, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, sourceExpires.UnixMilli(), rerolled.Expires.Int64)
}

// TestRerollRootKeyIgnoresLegacyKeys guarantees rootKeys.rerollKey manages only
// keys in the new root-key store.
func TestRerollRootKeyIgnoresLegacyKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	legacy := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: legacy.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
}

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
	h.Register(route)
	return route
}

func rootKeySession(t *testing.T, key string) *zen.Session {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	return session
}

func headers(bearer string) http.Header {
	return http.Header{"Authorization": {"Bearer " + bearer}, "Content-Type": {"application/json"}}
}

func call(h *testutil.Harness, route *handler.Handler, bearer string, req handler.Request) testutil.TestResponse[handler.Response] {
	return testutil.CallRoute[handler.Request, handler.Response](h, route, headers(bearer), req)
}

func countNewRootKeys(t *testing.T, h *testutil.Harness, workspaceID string) int {
	t.Helper()
	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM unkey_root_keys WHERE workspace_id = ?", workspaceID).Scan(&count))
	return count
}

// TestRerollRootKeyWaitsForOriginalRowLock guarantees rerolls serialize with
// other transactions that hold the original key row lock and copy the permissions
// that remain after the lock is released.
func TestRerollRootKeyWaitsForOriginalRowLock(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{permission}})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", permission)
	tx, err := h.DB.RW().Begin(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	var id string
	require.NoError(t, tx.QueryRowContext(t.Context(), "SELECT id FROM unkey_root_keys WHERE id = ? FOR UPDATE", source.KeyID).Scan(&id))

	done := make(chan testutil.TestResponse[handler.Response], 1)
	go func() {
		done <- call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	}()
	select {
	case res := <-done:
		t.Fatalf("reroll completed while original row was locked: %d %s", res.Status, res.RawBody)
	case <-time.After(200 * time.Millisecond):
	}
	_, err = tx.ExecContext(t.Context(), "DELETE FROM unkey_principal_permissions WHERE workspace_id = ? AND principal_type = 'root_key' AND principal_id = ?", workspace.ID, source.KeyID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	select {
	case res := <-done:
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		permissions, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
			WorkspaceID: workspace.ID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: res.Body.Data.KeyId,
		})
		require.NoError(t, err)
		require.Empty(t, permissions)
	case <-time.After(5 * time.Second):
		t.Fatal("reroll did not complete after row lock was released")
	}
}
