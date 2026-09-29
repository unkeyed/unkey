package handler_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_update_key"
)

// TestUpdateRootKeyDisablesWarmAuthentication guarantees disabling a root key
// immediately stops authentication, even after the root-key cache is warm.
func TestUpdateRootKeyDisablesWarmAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	session := rootKeySession(t, target.Key)
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)

	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+target.KeyID+"#write")
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Enabled: new(false)})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
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

// TestUpdateRootKeyReplacesLegacyAndNewPermissions guarantees a replacement
// removes legacy grants and stores only the validated URN permission set.
func TestUpdateRootKeyReplacesLegacyAndNewPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID,
		Permissions: []seed.CreatePermissionRequest{{WorkspaceID: h.Resources().RootWorkspace.ID, Name: "api.*.read_key", Slug: "api.*.read_key"}},
	})
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", permission)
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Permissions: &[]string{permission, permission}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	legacy, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: target.KeyID})
	require.NoError(t, err)
	require.Empty(t, legacy)
	stored, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
		ForWorkspaceID: workspace.ID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: target.KeyID,
	})
	require.NoError(t, err)
	require.Equal(t, []string{permission}, stored)
}

// TestUpdateRootKeyRejectsBroaderPermissionsWithoutWrites guarantees update
// cannot delegate permissions beyond the caller's own permissions.
func TestUpdateRootKeyRejectsBroaderPermissionsWithoutWrites(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Permissions: &[]string{"unkey:v1:" + workspace.ID + ":**#*"}})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	stored, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
		ForWorkspaceID: workspace.ID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: target.KeyID,
	})
	require.NoError(t, err)
	require.Empty(t, stored)
}

// TestExpiringCallerCannotGrantToLongerLivedRootKey guarantees updates cannot
// extend a delegated permission beyond the caller's own expiration.
func TestExpiringCallerCannotGrantToLongerLivedRootKey(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	expires := h.Clock.Now().Add(time.Hour)
	caller := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID, Expires: &expires,
		Permissions: []seed.CreatePermissionRequest{
			{WorkspaceID: h.Resources().RootWorkspace.ID, Name: "rootKeys/*#write", Slug: "unkey:v1:" + workspace.ID + ":rootKeys/*#write"},
			{WorkspaceID: h.Resources().RootWorkspace.ID, Name: permission, Slug: permission},
		},
	})
	res := call(h, route, caller.Key, handler.Request{KeyId: target.KeyID, Permissions: &[]string{permission}})
	require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
}

// TestUpdateRootKeyChangesNewAndLegacyRowsWithSameID guarantees migration twins
// receive the same name and enabled state.
func TestUpdateRootKeyChangesNewAndLegacyRowsWithSameID(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	legacy := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	require.NoError(t, db.Query.InsertUnkeyRootKey(t.Context(), h.DB.RW(), db.InsertUnkeyRootKeyParams{
		ID: legacy.KeyID, ForWorkspaceID: workspace.ID, Hash: hash.Sha256(legacy.Key), Name: sql.NullString{},
		Prefix: "unkey", Start: "test", End: "test", Enabled: true, Expires: sql.NullTime{}, CreatedAt: h.Clock.Now().UnixMilli(),
	}))
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: legacy.KeyID, Name: nullable.NewNullableWithValue("renamed"), Enabled: new(false)})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	var legacyName, newName string
	var legacyEnabled, newEnabled bool
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT name, enabled FROM `keys` WHERE id = ?", legacy.KeyID).Scan(&legacyName, &legacyEnabled))
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT name, enabled FROM unkey_root_keys WHERE id = ?", legacy.KeyID).Scan(&newName, &newEnabled))
	require.Equal(t, "renamed", legacyName)
	require.Equal(t, "renamed", newName)
	require.False(t, legacyEnabled)
	require.False(t, newEnabled)
}
