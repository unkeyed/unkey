package handler_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

// TestRootKeyAuthenticationPrefersNewStore guarantees migration overlap resolves
// to the new root-key row before the legacy fallback is queried.
func TestRootKeyAuthenticationPrefersNewStore(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	legacy := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID:    r.RootWorkspace.ID,
		KeySpaceID:     r.RootKeySpace.ID,
		ForWorkspaceID: &r.UserWorkspace.ID,
	})
	newID := uid.New(uid.KeyPrefix)
	require.NoError(t, db.Query.InsertUnkeyRootKey(t.Context(), h.DB.RW(), db.InsertUnkeyRootKeyParams{
		ID:          newID,
		WorkspaceID: r.UserWorkspace.ID,
		Hash:        hash.Sha256(legacy.Key),
		Name:        sql.NullString{},
		Prefix:      "unkey",
		Start:       "test",
		End:         "test",
		Enabled:     true,
		Expires:     sql.NullInt64{},
		CreatedAt:   1_700_000_000_000,
	}))

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+legacy.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))

	rootKey, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	require.Equal(t, newID, rootKey.Key.ID)

	_, err = db.Query.SoftDeleteUnkeyRootKey(t.Context(), h.DB.RW(), db.SoftDeleteUnkeyRootKeyParams{
		Now:         sql.NullInt64{Int64: 1, Valid: true},
		ID:          newID,
		WorkspaceID: r.UserWorkspace.ID,
	})
	require.NoError(t, err)
	h.Caches.RootKeyByHash.Remove(t.Context(), hash.Sha256(legacy.Key))
	for range 2 {
		_, err = h.Keys.GetRootKey(t.Context(), session)
		require.Error(t, err)
	}
}

// TestNewRootKeyAuthenticationIgnoresLegacyOwnership guarantees new root keys
// do not depend on the internal workspace, API, or keyspace used by legacy keys.
func TestNewRootKeyAuthenticationIgnoresLegacyOwnership(t *testing.T) {
	h, route, p := newHarness(t)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	r := h.Resources()
	_, err := db.Query.UpdateWorkspaceEnabled(t.Context(), h.DB.RW(), db.UpdateWorkspaceEnabledParams{Enabled: false, ID: r.RootWorkspace.ID})
	require.NoError(t, err)
	require.NoError(t, db.Query.SoftDeleteApi(t.Context(), h.DB.RW(), db.SoftDeleteApiParams{
		Now:   sql.NullInt64{Int64: 1, Valid: true},
		ApiID: r.RootApi.ID,
	}))
	_, err = h.DB.RW().ExecContext(t.Context(), "DELETE FROM key_auth WHERE id = ?", r.RootKeySpace.ID)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+res.Body.Data.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	rootKey, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	require.Equal(t, p.AuthorizedWorkspaceID, rootKey.AuthorizedWorkspaceID)
}

// TestNewRootKeyAuthenticationChecksLifecycle guarantees the new storage does not
// bypass lifecycle checks. For example, expired or disabled keys fail auth, as do
// keys whose target workspace is missing or disabled.
func TestNewRootKeyAuthenticationChecksLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(t *testing.T, h *testutil.Harness, keyID, workspaceID string)
		code   codes.URN
	}{
		{"disabled", func(t *testing.T, h *testutil.Harness, keyID, workspaceID string) {
			require.NoError(t, db.Query.UpdateUnkeyRootKey(t.Context(), h.DB.RW(), db.UpdateUnkeyRootKeyParams{
				EnabledSpecified: 1,
				Enabled:          sql.NullBool{Bool: false, Valid: true},
				ID:               keyID,
				WorkspaceID:      workspaceID,
			}))
		}, codes.Auth.Authorization.KeyDisabled.URN()},
		{"expired", func(t *testing.T, h *testutil.Harness, keyID, workspaceID string) {
			require.NoError(t, db.Query.UpdateUnkeyRootKeyExpiration(t.Context(), h.DB.RW(), db.UpdateUnkeyRootKeyExpirationParams{
				Expires:     sql.NullInt64{Int64: 946684800000, Valid: true},
				ID:          keyID,
				WorkspaceID: workspaceID,
			}))
		}, codes.Auth.Authorization.Forbidden.URN()},
		{"deleted", func(t *testing.T, h *testutil.Harness, keyID, workspaceID string) {
			_, err := db.Query.SoftDeleteUnkeyRootKey(t.Context(), h.DB.RW(), db.SoftDeleteUnkeyRootKeyParams{
				Now:         sql.NullInt64{Int64: 1, Valid: true},
				ID:          keyID,
				WorkspaceID: workspaceID,
			})
			require.NoError(t, err)
		}, codes.Auth.Authentication.KeyNotFound.URN()},
		{"missing target", func(t *testing.T, h *testutil.Harness, keyID, _ string) {
			_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE unkey_root_keys SET workspace_id = ? WHERE id = ?", uid.New(uid.WorkspacePrefix), keyID)
			require.NoError(t, err)
		}, codes.Data.Workspace.NotFound.URN()},
		{"disabled target", func(t *testing.T, h *testutil.Harness, _, workspaceID string) {
			_, err := db.Query.UpdateWorkspaceEnabled(t.Context(), h.DB.RW(), db.UpdateWorkspaceEnabledParams{Enabled: false, ID: workspaceID})
			require.NoError(t, err)
		}, codes.Auth.Authorization.WorkspaceDisabled.URN()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, route, p := newHarness(t)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{}})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			tt.mutate(t, h, res.Body.Data.KeyId, p.AuthorizedWorkspaceID)
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.Header.Set("Authorization", "Bearer "+res.Body.Data.Key)
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
			for range 2 {
				_, err := h.Keys.GetRootKey(t.Context(), session)
				require.Error(t, err)
				code, ok := fault.GetCode(err)
				require.True(t, ok)
				require.Equal(t, tt.code, code)
			}
		})
	}
}
