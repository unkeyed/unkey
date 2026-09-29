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

// TestRerollLegacyRootKeyMigratesPermissionsAndRevokesOriginal guarantees a
// legacy root key can be rotated into the new store without losing access, and
// that expiration 0 revokes the original even when its cache entry is warm.
func TestRerollLegacyRootKeyMigratesPermissionsAndRevokesOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	source := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID,
		Permissions: []seed.CreatePermissionRequest{{WorkspaceID: h.Resources().RootWorkspace.ID, Name: permission, Slug: permission}},
		Roles: []seed.CreateRoleRequest{{WorkspaceID: h.Resources().RootWorkspace.ID, Name: "reader", Permissions: []seed.CreatePermissionRequest{
			{WorkspaceID: h.Resources().RootWorkspace.ID, Name: "api.*.read_key", Slug: "api.*.read_key"},
		}}},
	})
	sourceSession := rootKeySession(t, source.Key)
	_, err := h.Keys.GetRootKey(t.Context(), sourceSession)
	require.NoError(t, err)

	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", permission, "api.*.read_key")
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue[int64](0)})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.NotEqual(t, source.KeyID, res.Body.Data.KeyId)

	_, err = h.Keys.GetRootKey(t.Context(), sourceSession)
	require.Error(t, err)
	rerolled, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, res.Body.Data.Key))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"api.*.read_key", permission}, rerolled.Permissions)
	stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, workspace.ID, stored.WorkspaceID)
}

// TestRerollRootKeyRejectsPermissionEscalationWithoutWrites guarantees a
// caller cannot receive a secret for permissions it does not already hold.
func TestRerollRootKeyRejectsPermissionEscalationWithoutWrites(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	admin := "unkey:v1:" + workspace.ID + ":**#*"
	source := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID,
		Permissions: []seed.CreatePermissionRequest{{WorkspaceID: h.Resources().RootWorkspace.ID, Name: admin, Slug: admin}},
	})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	before := countNewRootKeys(t, h, workspace.ID)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue[int64](0)})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, before, countNewRootKeys(t, h, workspace.ID))
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, source.Key))
	require.NoError(t, err)
}

// TestRerollRootKeyOverlapCannotExtendOriginal guarantees the grace period
// keeps the original key usable, but never extends its existing expiration.
func TestRerollRootKeyOverlapCannotExtendOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	expires := h.Clock.Now().Add(time.Hour).Truncate(time.Millisecond)
	source := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID, Expires: &expires})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue((24 * time.Hour).Milliseconds())})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, source.Key))
	require.NoError(t, err)
	var got time.Time
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT expires FROM `keys` WHERE id = ?", source.KeyID).Scan(&got))
	require.Equal(t, expires.UnixMilli(), got.UnixMilli())
	stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, expires.UnixMilli(), stored.Expires.Time.UnixMilli())
}

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, KeyCache: h.Caches.VerificationKeyByHash, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
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

func call(h *testutil.Harness, route *handler.Handler, bearer string, req handler.Request) testutil.TestResponse[handler.Response] {
	return testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer " + bearer}, "Content-Type": {"application/json"}}, req)
}

func countNewRootKeys(t *testing.T, h *testutil.Harness, workspaceID string) int {
	t.Helper()
	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM unkey_root_keys WHERE workspace_id = ?", workspaceID).Scan(&count))
	return count
}

// TestRerollRootKeyRequiresExplicitNullToKeepOriginal guarantees callers must
// choose whether the original survives, and null keeps it without an expiry.
func TestRerollRootKeyRequiresExplicitNullToKeepOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	source := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	headers := http.Header{"Authorization": {"Bearer " + caller}, "Content-Type": {"application/json"}}

	missing := testutil.CallRoute[map[string]any, handler.Response](h, route, headers, map[string]any{"keyId": source.KeyID})
	require.Equal(t, http.StatusBadRequest, missing.Status, "%s", missing.RawBody)

	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, source.Key))
	require.NoError(t, err)
	var expires *time.Time
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT expires FROM `keys` WHERE id = ?", source.KeyID).Scan(&expires))
	require.Nil(t, expires)
}
