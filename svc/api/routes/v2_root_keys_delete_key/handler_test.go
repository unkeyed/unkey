package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_delete_key"
)

// TestDeleteRootKeyRevokesWarmAuthentication guarantees deleting a new root key
// immediately stops authentication, even after the root-key cache is warm.
func TestDeleteRootKeyRevokesWarmAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	session := rootKeySession(t, target.Key)
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+target.KeyID+"#delete")
	res := call(h, route, caller, handler.Request{KeyId: target.KeyID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
}

// TestDeleteRootKeyRecordsRootKeyAuditEvent guarantees root-key mutations are
// distinguishable from API-key mutations in the audit log.
func TestDeleteRootKeyRecordsRootKeyAuditEvent(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+target.KeyID+"#delete")

	res := call(h, route, caller, handler.Request{KeyId: target.KeyID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	var event string
	err := h.DB.RO().QueryRowContext(t.Context(),
		"SELECT JSON_UNQUOTE(JSON_EXTRACT(payload, '$.event')) FROM clickhouse_outbox WHERE workspace_id = ? ORDER BY pk DESC LIMIT 1",
		workspace.ID,
	).Scan(&event)
	require.NoError(t, err)
	require.Equal(t, "rootKey.delete", event)
}

// TestDeleteRootKeyAuditFallsBackToID guarantees an unnamed key remains
// identifiable in the audit log by its stable ID.
func TestDeleteRootKeyAuditFallsBackToID(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+target.KeyID+"#delete")

	res := call(h, route, caller, handler.Request{KeyId: target.KeyID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	var name string
	err := h.DB.RO().QueryRowContext(t.Context(),
		"SELECT JSON_UNQUOTE(JSON_EXTRACT(payload, '$.targets[0].name')) FROM clickhouse_outbox WHERE workspace_id = ? ORDER BY pk DESC LIMIT 1",
		workspace.ID,
	).Scan(&name)
	require.NoError(t, err)
	require.Equal(t, target.KeyID, name)
}

// TestDeleteRootKeyRequiresConcreteDeletePermission guarantees read, write, or
// another key's delete permission cannot revoke a root key.
func TestDeleteRootKeyRequiresConcreteDeletePermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	for _, permission := range []string{
		"unkey:v1:" + workspace.ID + ":rootKeys/" + target.KeyID + "#read",
		"unkey:v1:" + workspace.ID + ":rootKeys/" + target.KeyID + "#write",
		"unkey:v1:" + workspace.ID + ":rootKeys/key_other#delete",
	} {
		t.Run(permission, func(t *testing.T) {
			res := call(h, route, h.CreateRootKey(workspace.ID, permission), handler.Request{KeyId: target.KeyID})
			require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
		})
	}
}

// TestDeleteRootKeyIgnoresLegacyKeys guarantees rootKeys.deleteKey manages only
// keys in the new root-key store.
func TestDeleteRootKeyIgnoresLegacyKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	legacy := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#delete")
	res := call(h, route, caller, handler.Request{KeyId: legacy.KeyID})
	require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, legacy.Key))
	require.NoError(t, err)
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
