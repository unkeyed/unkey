package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_update_key"
)

// TestUpdateRootKeyDisablesWarmAuthentication guarantees disabling a new root
// key immediately stops authentication, even after the root-key cache is warm.
func TestUpdateRootKeyDisablesWarmAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	session := rootKeySession(t, target.Key)
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+target.KeyID+"#write")
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Name: nullable.NewNullableWithValue("renamed"), Enabled: new(false)})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
	stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), target.KeyID)
	require.NoError(t, err)
	require.Equal(t, "renamed", stored.Name.String)
}

// TestUpdateRootKeyReplacesPermissions guarantees a permission update replaces
// the complete principal permission set.
func TestUpdateRootKeyReplacesPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	oldPermission := "unkey:v1:" + workspace.ID + ":rootKeys/*#delete"
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{oldPermission}})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", permission)
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Permissions: &[]string{permission, permission}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Equal(t, []string{permission}, storedPermissions(t, h, workspace.ID, target.KeyID))
}

// TestUpdateRootKeyAcceptsMaximumPermissionLength guarantees the API accepts
// permission strings up to the storage limit of 512 characters.
func TestUpdateRootKeyAcceptsMaximumPermissionLength(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	prefix := "unkey:v1:" + workspace.ID + ":projects/"
	permission := prefix + strings.Repeat("p", 512-len(prefix)-len("#read")) + "#read"
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", permission)

	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Permissions: &[]string{permission}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Equal(t, []string{permission}, storedPermissions(t, h, workspace.ID, target.KeyID))
}

// TestUpdateRootKeyRejectsBroaderPermissionsWithoutWrites guarantees update
// cannot delegate permissions beyond the caller's own permissions.
func TestUpdateRootKeyRejectsBroaderPermissionsWithoutWrites(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Permissions: &[]string{"unkey:v1:" + workspace.ID + ":**#*"}})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Empty(t, storedPermissions(t, h, workspace.ID, target.KeyID))
}

// TestExpiringCallerCannotUpdateLongerLivedRootKey guarantees updates cannot
// grant permissions to a key that outlives the caller.
func TestExpiringCallerCannotUpdateLongerLivedRootKey(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	expires := h.Clock.Now().Add(time.Hour)
	caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID, Expires: &expires,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#write", permission},
	})
	res := call(h, route, caller.Key, handler.Request{KeyId: target.KeyID, Permissions: &[]string{permission}})
	require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
}

// TestUpdateRootKeyIgnoresLegacyKeys guarantees rootKeys.updateKey manages only
// keys in the new root-key store.
func TestUpdateRootKeyIgnoresLegacyKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	legacy := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: legacy.KeyID, Enabled: new(false)})
	require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
	var enabled bool
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT enabled FROM `keys` WHERE id = ?", legacy.KeyID).Scan(&enabled))
	require.True(t, enabled)
}

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, KeyCache: h.Caches.VerificationKeyByHash, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
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

func storedPermissions(t *testing.T, h *testutil.Harness, workspaceID, keyID string) []string {
	t.Helper()
	permissions, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
		WorkspaceID: workspaceID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: keyID,
	})
	require.NoError(t, err)
	return permissions
}
