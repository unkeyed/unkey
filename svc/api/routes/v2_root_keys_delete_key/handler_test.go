package handler_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_delete_key"
)

// TestDeleteRootKeyRevokesWarmAuthentication guarantees deleting a legacy root
// key immediately stops authentication, even after the root-key cache is warm.
func TestDeleteRootKeyRevokesWarmAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, KeyCache: h.Caches.VerificationKeyByHash, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	target := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+target.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)

	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+target.KeyID+"#delete")
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer " + caller}, "Content-Type": {"application/json"}}, handler.Request{KeyId: target.KeyID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
}

// TestDeleteRootKeyRevokesNewAndLegacyRowsWithSameID guarantees migration twins
// cannot remain active after deletion of the shared root-key ID.
func TestDeleteRootKeyRevokesNewAndLegacyRowsWithSameID(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, KeyCache: h.Caches.VerificationKeyByHash, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	legacy := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	require.NoError(t, db.Query.InsertUnkeyRootKey(t.Context(), h.DB.RW(), db.InsertUnkeyRootKeyParams{
		ID:          legacy.KeyID,
		WorkspaceID: workspace.ID,
		Hash:        hash.Sha256(legacy.Key),
		Name:        sql.NullString{},
		Prefix:      "unkey",
		Start:       "test",
		End:         "test",
		Enabled:     true,
		Expires:     sql.NullTime{},
		CreatedAt:   h.Clock.Now().UnixMilli(),
	}))
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#delete")
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer " + caller}, "Content-Type": {"application/json"}}, handler.Request{KeyId: legacy.KeyID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+legacy.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
	var legacyDeleted, newDeleted bool
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT deleted_at_m IS NOT NULL FROM `keys` WHERE id = ?", legacy.KeyID).Scan(&legacyDeleted))
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT deleted_at IS NOT NULL FROM unkey_root_keys WHERE id = ?", legacy.KeyID).Scan(&newDeleted))
	require.True(t, legacyDeleted)
	require.True(t, newDeleted)
}

// TestDeleteRootKeyRequiresConcreteDeletePermission guarantees read, write, or
// another key's delete permission cannot revoke a root key.
func TestDeleteRootKeyRequiresConcreteDeletePermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, KeyCache: h.Caches.VerificationKeyByHash, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	target := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	for _, permission := range []string{
		"unkey:v1:" + workspace.ID + ":rootKeys/" + target.KeyID + "#read",
		"unkey:v1:" + workspace.ID + ":rootKeys/" + target.KeyID + "#write",
		"unkey:v1:" + workspace.ID + ":rootKeys/key_other#delete",
	} {
		t.Run(permission, func(t *testing.T) {
			caller := h.CreateRootKey(workspace.ID, permission)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer " + caller}, "Content-Type": {"application/json"}}, handler.Request{KeyId: target.KeyID})
			require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
		})
	}
}
