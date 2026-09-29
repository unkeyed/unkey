package handler_test

import (
	"database/sql"
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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_list_keys"
)

// TestListRootKeysPaginatesBothStores guarantees individually readable keys from
// both stores share one pagination stream. For example, denied keys between two
// readable keys neither shorten the page nor appear in its cursor.
func TestListRootKeysPaginatesBothStores(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	workspace := h.CreateWorkspace()
	ids := make([]string, 8)
	for i := range ids {
		ids[i] = strings.ToLower(uid.New(uid.KeyPrefix))
	}
	slices.Sort(ids)
	for i, id := range ids {
		if i%2 == 0 {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID:    h.Resources().RootWorkspace.ID,
				KeySpaceID:     h.Resources().RootKeySpace.ID,
				ForWorkspaceID: &workspace.ID,
			})
			_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE `keys` SET id = ? WHERE id = ?", id, key.KeyID)
			require.NoError(t, err)
		} else {
			require.NoError(t, db.Query.InsertUnkeyRootKey(t.Context(), h.DB.RW(), db.InsertUnkeyRootKeyParams{
				ID:             id,
				WorkspaceID:    h.Resources().RootWorkspace.ID,
				KeyAuthID:      h.Resources().RootKeySpace.ID,
				ForWorkspaceID: workspace.ID,
				Hash:           uid.New("hash"),
				Name:           sql.NullString{},
				Prefix:         "unkey",
				Start:          "visible",
				End:            "tail",
				Enabled:        true,
				Expires:        sql.NullTime{},
				CreatedAt:      1700000000000,
			}))
		}
	}
	caller := h.CreateRootKey(workspace.ID,
		"unkey:v1:"+workspace.ID+":rootKeys/"+ids[3]+"#read",
		"unkey:v1:"+workspace.ID+":rootKeys/"+ids[6]+"#read",
	)
	headers := http.Header{"Authorization": {"Bearer " + caller}, "Content-Type": {"application/json"}}
	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Limit: new(1)})
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Len(t, first.Body.Data, 1)
	require.Equal(t, ids[3], first.Body.Data[0].KeyId)
	require.True(t, first.Body.Pagination.HasMore)
	require.Equal(t, new(ids[6]), first.Body.Pagination.Cursor)
	last := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Limit: new(1), Cursor: first.Body.Pagination.Cursor})
	require.Equal(t, http.StatusOK, last.Status, "%s", last.RawBody)
	require.Len(t, last.Body.Data, 1)
	require.Equal(t, ids[6], last.Body.Data[0].KeyId)
	require.False(t, last.Body.Pagination.HasMore)
	require.Nil(t, last.Body.Pagination.Cursor)
	for _, i := range []int{0, 1, 2, 4, 5, 7} {
		require.NotContains(t, first.RawBody, ids[i])
		require.NotContains(t, last.RawBody, ids[i])
	}
}

// TestListRootKeysReturnsEffectivePermissions guarantees metadata and permission
// strings survive listing without exposing secrets. For example, a legacy role
// permission is included, duplicate direct permissions appear once, and an expired,
// disabled key in the new table remains visible with its exact expiry timestamp.
func TestListRootKeysReturnsEffectivePermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	workspace := h.CreateWorkspace()
	legacy := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID:    h.Resources().RootWorkspace.ID,
		KeySpaceID:     h.Resources().RootKeySpace.ID,
		ForWorkspaceID: &workspace.ID,
		Permissions:    []seed.CreatePermissionRequest{{WorkspaceID: h.Resources().RootWorkspace.ID, Name: "direct", Slug: "api.*.read_key"}},
		Roles: []seed.CreateRoleRequest{{WorkspaceID: h.Resources().RootWorkspace.ID, Name: "reader", Permissions: []seed.CreatePermissionRequest{
			{WorkspaceID: h.Resources().RootWorkspace.ID, Name: "inherited", Slug: "api.*.delete_key"},
		}}},
	})
	_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE `keys` SET name = NULL WHERE id = ?", legacy.KeyID)
	require.NoError(t, err)
	id := uid.New(uid.KeyPrefix)
	require.NoError(t, db.Query.InsertUnkeyRootKey(t.Context(), h.DB.RW(), db.InsertUnkeyRootKeyParams{
		ID:             id,
		WorkspaceID:    h.Resources().RootWorkspace.ID,
		KeyAuthID:      h.Resources().RootKeySpace.ID,
		ForWorkspaceID: workspace.ID,
		Hash:           "never-return-this-hash",
		Name:           sql.NullString{String: "", Valid: true},
		Prefix:         "unkey",
		Start:          "display",
		End:            "tail",
		Enabled:        false,
		Expires:        sql.NullTime{Time: time.UnixMilli(1600000000123), Valid: true},
		CreatedAt:      1500000000456,
	}))
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	for _, row := range []struct {
		workspace  string
		kind       db.UnkeyPrincipalPermissionsPrincipalType
		id         string
		permission string
	}{
		{workspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, legacy.KeyID, "api.*.read_key"},
		{workspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, id, permission},
		{workspace.ID, db.UnkeyPrincipalPermissionsPrincipalTypeOidc, id, "wrong-type"},
		{h.CreateWorkspace().ID, db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, id, "wrong-workspace"},
	} {
		require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), h.DB.RW(), db.InsertUnkeyPermissionParams{
			ID:             uid.New(uid.PermissionPrefix),
			ForWorkspaceID: row.workspace,
			PrincipalType:  row.kind,
			PrincipalID:    row.id,
			Slug:           row.permission,
			CreatedAt:      1500000000456,
		}))
	}
	caller := h.CreateRootKey(workspace.ID,
		"unkey:v1:"+workspace.ID+":rootKeys/"+legacy.KeyID+"#read",
		"unkey:v1:"+workspace.ID+":rootKeys/"+id+"#read",
	)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer " + caller}, "Content-Type": {"application/json"},
	}, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Len(t, res.Body.Data, 2)
	for _, item := range res.Body.Data {
		if item.KeyId == legacy.KeyID {
			require.Equal(t, []string{"api.*.delete_key", "api.*.read_key"}, item.Permissions)
			require.True(t, item.Name.IsNull())
			require.True(t, item.Expires.IsNull())
		} else {
			require.Equal(t, id, item.KeyId)
			require.Equal(t, []string{permission}, item.Permissions)
			require.Equal(t, "", item.Name.MustGet())
			require.Equal(t, int64(1600000000123), item.Expires.MustGet())
			require.Equal(t, int64(1500000000456), item.CreatedAt)
			require.Equal(t, "unkey_display", item.Start)
			require.Equal(t, "tail", item.End)
			require.False(t, item.Enabled)
		}
	}
	require.NotContains(t, res.RawBody, legacy.Key)
	require.NotContains(t, res.RawBody, "never-return-this-hash")
	require.NotContains(t, res.RawBody, `"hash"`)
	require.NotContains(t, res.RawBody, `"key"`)
}

// TestListRootKeysValidatesRequestsAndReadAccess guarantees authentication and
// request scope cannot be bypassed. For example, workspace input and limit 101
// are rejected, while write-only or foreign-workspace permissions reveal no keys.
func TestListRootKeysValidatesRequestsAndReadAccess(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	workspace := h.CreateWorkspace()
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
		{"maximum limit", read, map[string]any{"limit": 100}, 200},
		{"write only", "unkey:v1:" + workspace.ID + ":rootKeys/*#write", map[string]any{}, 200},
		{"legacy only", "api.*.read_key", map[string]any{}, 200},
		{"legacy star", "*", map[string]any{}, 200},
		{"foreign workspace", "unkey:v1:ws_other:rootKeys/*#read", map[string]any{}, 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			caller := h.CreateRootKey(workspace.ID, tt.permission)
			res := testutil.CallRoute[map[string]any, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer " + caller}, "Content-Type": {"application/json"},
			}, tt.body)
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			if tt.status == 200 && tt.permission != read {
				require.Empty(t, res.Body.Data)
				require.False(t, res.Body.Pagination.HasMore)
				require.Nil(t, res.Body.Pagination.Cursor)
				require.Contains(t, res.RawBody, `"data":[]`)
			}
		})
	}
	for _, tt := range []struct {
		bearer string
		status int
	}{
		{"", http.StatusBadRequest},
		{"Bearer invalid", http.StatusUnauthorized},
	} {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
			"Authorization": {tt.bearer}, "Content-Type": {"application/json"},
		}, handler.Request{})
		require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
	}
}

// TestListRootKeysExcludesForeignAndDeletedKeys guarantees a workspace admin
// sees only its root-key inventory. For example, ordinary API keys and deleted
// keys in either store stay hidden even when the caller has workspace-wide access.
func TestListRootKeysExcludesForeignAndDeletedKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	workspace := h.CreateWorkspace()
	foreign := h.CreateWorkspace()
	caller := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID:    h.Resources().RootWorkspace.ID,
		KeySpaceID:     h.Resources().RootKeySpace.ID,
		ForWorkspaceID: &workspace.ID,
		Permissions: []seed.CreatePermissionRequest{{
			WorkspaceID: h.Resources().RootWorkspace.ID,
			Name:        "admin",
			Slug:        "unkey:v1:" + workspace.ID + ":**#*",
		}},
	})
	for _, req := range []seed.CreateKeyRequest{
		{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID},
		{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &foreign.ID},
		{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID, Deleted: true},
	} {
		h.CreateKey(req)
	}
	for _, target := range []string{workspace.ID, foreign.ID} {
		id := uid.New(uid.KeyPrefix)
		require.NoError(t, db.Query.InsertUnkeyRootKey(t.Context(), h.DB.RW(), db.InsertUnkeyRootKeyParams{
			ID:             id,
			WorkspaceID:    h.Resources().RootWorkspace.ID,
			KeyAuthID:      h.Resources().RootKeySpace.ID,
			ForWorkspaceID: target,
			Hash:           uid.New("hash"),
			Name:           sql.NullString{},
			Prefix:         "unkey",
			Start:          "hidden",
			End:            "tail",
			Enabled:        true,
			Expires:        sql.NullTime{},
			CreatedAt:      1700000000000,
		}))
		if target == workspace.ID {
			_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE unkey_root_keys SET deleted_at = 1 WHERE id = ?", id)
			require.NoError(t, err)
		}
	}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer " + caller.Key}, "Content-Type": {"application/json"},
	}, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, caller.KeyID, res.Body.Data[0].KeyId)
}
