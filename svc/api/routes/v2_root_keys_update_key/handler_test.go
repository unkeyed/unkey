package handler_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_update_key"
)

func TestUpdateRootKeyRejectsMissingCallerWithExistingTarget(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#write"},
	})
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, caller.Key))
	require.NoError(t, err)
	_, err = db.Query.SoftDeleteUnkeyRootKey(t.Context(), h.DB.RW(), db.SoftDeleteUnkeyRootKeyParams{
		ID:          caller.KeyID,
		WorkspaceID: workspace.ID,
		Now:         sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: true},
	})
	require.NoError(t, err)
	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer " + caller.Key},
		"Content-Type":  {"application/json"},
	}, handler.Request{KeyId: target.KeyID, Enabled: new(false)})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, codes.Auth.Authorization.Forbidden.DocsURL(), res.Body.Error.Type)
	require.Equal(t, "The calling root key is no longer available.", res.Body.Error.Detail)
	stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), target.KeyID)
	require.NoError(t, err)
	require.True(t, stored.Enabled)
}

func TestCachedRootKeyCannotRemoveNewRestriction(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	require.NoError(t, db.Query.UpdateRootKeyIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateRootKeyIPAllowlistEnabledParams{
		WorkspaceID: workspace.ID,
		Enabled:     true,
	}))
	caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#write"},
	})
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, caller.Key))
	require.NoError(t, err)
	// Another API instance can tighten this list without evicting our local cache.
	require.NoError(t, db.Query.UpdateUnkeyRootKey(t.Context(), h.DB.RW(), db.UpdateUnkeyRootKeyParams{
		ID:                   caller.KeyID,
		WorkspaceID:          workspace.ID,
		IpAllowlistSpecified: 1,
		IpAllowlist:          []string{"192.0.2.0/25"},
	}))
	res := call(h, route, caller.Key, handler.Request{KeyId: caller.KeyID, IpAllowlist: &[]string{}})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Contains(t, string(res.RawBody), "must stay within the calling root key's allowed networks")
	stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RW(), caller.KeyID)
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.0/25"}, []string(stored.IpAllowlist))
}

func TestRestrictedRootKeyCannotWidenNetworksOnUpdate(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	require.NoError(t, db.Query.UpdateRootKeyIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateRootKeyIPAllowlistEnabledParams{WorkspaceID: workspace.ID, Enabled: true}))
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#write"
	admin := h.CreateRootKey(workspace.ID, permission)
	caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{permission}})
	res := call(h, route, admin, handler.Request{KeyId: caller.KeyID, IpAllowlist: &[]string{"192.0.2.0/24"}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	for _, test := range []struct {
		name   string
		before []string
		after  *[]string
		status int
	}{
		{"clear", []string{"192.0.2.0/25"}, &[]string{}, http.StatusForbidden},
		{"widen target within caller", []string{"192.0.2.0/25"}, &[]string{"192.0.2.0/24"}, http.StatusOK},
		{"escape caller", []string{"192.0.2.0/25"}, &[]string{"192.0.0.0/16"}, http.StatusForbidden},
		{"modify unrestricted target", nil, nil, http.StatusForbidden},
		{"preserve", []string{"192.0.2.0/25"}, nil, http.StatusOK},
		{"narrow", []string{"192.0.2.0/25"}, &[]string{"192.0.2.0/26"}, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
			res := call(h, route, admin, handler.Request{KeyId: target.KeyID, IpAllowlist: new(append([]string{}, test.before...))})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			res = call(h, route, caller.Key, handler.Request{KeyId: target.KeyID, Name: nullable.NewNullableWithValue("changed"), IpAllowlist: test.after})
			require.Equal(t, test.status, res.Status, "%s", res.RawBody)
			if test.status != http.StatusOK {
				stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), target.KeyID)
				require.NoError(t, err)
				require.NotEqual(t, "changed", stored.Name.String)
				require.ElementsMatch(t, test.before, stored.IpAllowlist)
			} else if test.after != nil {
				stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), target.KeyID)
				require.NoError(t, err)
				require.ElementsMatch(t, *test.after, stored.IpAllowlist)
			}
		})
	}
	res = call(h, route, caller.Key, handler.Request{KeyId: caller.KeyID, IpAllowlist: &[]string{}})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
}

func TestUpdateRootKeyRestrictsWarmAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	session := rootKeySession(t, target.Key)
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, IpAllowlist: &[]string{"203.0.113.129/25"}})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.NoError(t, db.Query.UpdateRootKeyIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateRootKeyIPAllowlistEnabledParams{WorkspaceID: workspace.ID, Enabled: true}))
	res = call(h, route, caller, handler.Request{KeyId: target.KeyID, IpAllowlist: &[]string{"203.0.113.129/25"}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.NoError(t, db.Query.UpdateRootKeyIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateRootKeyIPAllowlistEnabledParams{WorkspaceID: workspace.ID, Enabled: false}))
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
	session.SetClientIP(netip.MustParseAddr("203.0.113.200"))
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	res = call(h, route, caller, handler.Request{KeyId: target.KeyID, Name: nullable.NewNullableWithValue("renamed")})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	session.SetClientIP(netip.MustParseAddr("203.0.113.127"))
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
	res = call(h, route, caller, handler.Request{KeyId: target.KeyID, IpAllowlist: &[]string{}})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
	require.NoError(t, db.Query.UpdateRootKeyIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateRootKeyIPAllowlistEnabledParams{WorkspaceID: workspace.ID, Enabled: true}))
	res = call(h, route, caller, handler.Request{KeyId: target.KeyID, IpAllowlist: &[]string{}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
}

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

// TestUpdateRootKeyAuditsRemovedPermissions guarantees replacing permissions
// records removals without recording retained permissions as new grants.
func TestUpdateRootKeyAuditsRemovedPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	removedPermission := "unkey:v1:" + workspace.ID + ":rootKeys/*#delete"
	retainedPermission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{removedPermission, retainedPermission}})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", retainedPermission)

	res := call(h, route, caller, handler.Request{KeyId: target.KeyID, Permissions: &[]string{retainedPermission}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Equal(t, []string{retainedPermission}, storedPermissions(t, h, workspace.ID, target.KeyID))

	logs := h.FindAuditLogsByTargetID(t.Context(), t, target.KeyID)
	require.Len(t, logs, 2)
	var disconnect *auditlog.Event
	for i := range logs {
		require.NotEqual(t, string(auditlog.AuthConnectPermissionKeyEvent), logs[i].Event)
		if logs[i].Event == string(auditlog.AuthDisconnectPermissionKeyEvent) {
			disconnect = &logs[i]
			break
		}
	}
	require.NotNil(t, disconnect)
	require.Equal(t, "Removed permission "+removedPermission+" from root key "+target.KeyID, disconnect.Description)
	require.Len(t, disconnect.Targets, 2)
	require.Equal(t, removedPermission, disconnect.Targets[1].Name)
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
	stored, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), legacy.KeyID)
	require.NoError(t, err)
	require.True(t, stored.Enabled)
}

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
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
