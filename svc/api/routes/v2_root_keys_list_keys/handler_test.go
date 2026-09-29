package handler_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_list_keys"
)

// TestListRootKeysPaginatesReadableNewKeys guarantees denied keys neither
// shorten a page nor appear in its cursor.
func TestListRootKeysPaginatesReadableNewKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.CreateWorkspace()
	ids := make([]string, 6)
	for i := range ids {
		ids[i] = strings.ToLower(uid.New(uid.KeyPrefix))
	}
	slices.Sort(ids)
	for _, id := range ids {
		key := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
		_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE unkey_root_keys SET id = ? WHERE id = ?", id, key.KeyID)
		require.NoError(t, err)
	}
	caller := h.CreateRootKey(workspace.ID,
		"unkey:v1:"+workspace.ID+":rootKeys/"+ids[2]+"#read",
		"unkey:v1:"+workspace.ID+":rootKeys/"+ids[4]+"#read",
	)
	first := call(h, route, caller, handler.Request{Limit: new(1)})
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Equal(t, []string{ids[2]}, keyIDs(first.Body.Data))
	require.Equal(t, new(ids[4]), first.Body.Pagination.Cursor)
	last := call(h, route, caller, handler.Request{Limit: new(1), Cursor: first.Body.Pagination.Cursor})
	require.Equal(t, http.StatusOK, last.Status, "%s", last.RawBody)
	require.Equal(t, []string{ids[4]}, keyIDs(last.Body.Data))
	require.False(t, last.Body.Pagination.HasMore)
}

// TestListRootKeysReturnsNewMetadataAndPermissions guarantees listing returns
// principal permissions and metadata without exposing secrets.
func TestListRootKeysReturnsNewMetadataAndPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.CreateWorkspace()
	expires := time.UnixMilli(1600000000123)
	name := ""
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	key := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID, Name: &name, Disabled: true, Expires: &expires, Permissions: []string{permission},
	})
	for _, row := range []struct {
		workspaceID   string
		principalType db.UnkeyPrincipalPermissionsPrincipalType
		slug          string
	}{
		{workspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeOidc, "wrong-type"},
		{h.CreateWorkspace().ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, "wrong-workspace"},
	} {
		require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), db.InsertUnkeyPermissionParams{
			ID: uid.New(uid.PermissionPrefix), WorkspaceID: row.workspaceID, PrincipalType: row.principalType,
			PrincipalID: key.KeyID, Slug: row.slug, CreatedAt: 1,
		}))
	}
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+key.KeyID+"#read")
	res := call(h, route, caller, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	item := res.Body.Data[0]
	require.Equal(t, key.KeyID, item.KeyId)
	require.Equal(t, []string{permission}, item.Permissions)
	require.Equal(t, "", item.Name.MustGet())
	require.Equal(t, expires.UnixMilli(), item.Expires.MustGet())
	require.False(t, item.Enabled)
	require.NotContains(t, res.RawBody, key.Key)
	require.NotContains(t, res.RawBody, `"hash"`)
}

// TestListRootKeysIgnoresLegacyForeignAndDeletedKeys guarantees the route lists
// only live keys in the new store for the authenticated customer workspace.
func TestListRootKeysIgnoresLegacyForeignAndDeletedKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.CreateWorkspace()
	foreign := h.CreateWorkspace()
	visible := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	deleted := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE unkey_root_keys SET deleted_at = 1 WHERE id = ?", deleted.KeyID)
	require.NoError(t, err)
	h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: foreign.ID})
	legacy := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#read")
	res := call(h, route, caller, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Equal(t, []string{visible.KeyID}, keyIDs(res.Body.Data))
	require.NotContains(t, res.RawBody, legacy.KeyID)
}

// TestListRootKeysValidatesRequestsAndReadAccess guarantees request validation
// and read permissions cannot be bypassed.
func TestListRootKeysValidatesRequestsAndReadAccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.CreateWorkspace()
	h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	read := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	for _, tt := range []struct {
		name       string
		permission string
		body       map[string]any
		status     int
	}{
		{"limit zero", read, map[string]any{"limit": 0}, 400},
		{"limit too high", read, map[string]any{"limit": 101}, 400},
		{"workspace input", read, map[string]any{"workspaceId": workspace.ID}, 400},
		{"invalid cursor", read, map[string]any{"cursor": 42}, 400},
		{"write only", "unkey:v1:" + workspace.ID + ":rootKeys/*#write", map[string]any{}, 200},
		{"legacy only", "api.*.read_key", map[string]any{}, 200},
		{"foreign workspace", "unkey:v1:ws_other:rootKeys/*#read", map[string]any{}, 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			caller := h.CreateRootKey(workspace.ID, tt.permission)
			res := testutil.CallRoute[map[string]any, handler.Response](h, route, headers(caller), tt.body)
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			if tt.status == 200 {
				require.Empty(t, res.Body.Data)
				require.Contains(t, res.RawBody, `"data":[]`)
			}
		})
	}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers("invalid"), handler.Request{})
	require.Equal(t, http.StatusUnauthorized, res.Status, "%s", res.RawBody)
}

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	return route
}

func headers(bearer string) http.Header {
	return http.Header{"Authorization": {"Bearer " + bearer}, "Content-Type": {"application/json"}}
}

func call(h *testutil.Harness, route *handler.Handler, bearer string, req handler.Request) testutil.TestResponse[handler.Response] {
	return testutil.CallRoute[handler.Request, handler.Response](h, route, headers(bearer), req)
}

func keyIDs(items []openapi.V2RootKeysListKeysResponseData) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.KeyId)
	}
	return ids
}
