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
	updatekey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
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
		ID:             newID,
		ForWorkspaceID: r.UserWorkspace.ID,
		Hash:           hash.Sha256(legacy.Key),
		Name:           sql.NullString{},
		Prefix:         "unkey",
		Start:          "test",
		End:            "test",
		Enabled:        true,
		Expires:        sql.NullTime{},
		CreatedAt:      1_700_000_000_000,
	}))

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+legacy.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))

	rootKey, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	require.Equal(t, newID, rootKey.Key.ID)
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
	_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE workspaces SET enabled = FALSE WHERE id = ?", r.RootWorkspace.ID)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "UPDATE apis SET deleted_at_m = 1 WHERE key_auth_id = ?", r.RootKeySpace.ID)
	require.NoError(t, err)
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

// TestLegacyRootKeyDisableInvalidatesAuthentication guarantees an authorized
// update removes cached root authentication. For example, a legacy root key
// authenticated before it is disabled cannot continue using its warm cache entry.
func TestLegacyRootKeyDisableInvalidatesAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID:    r.RootWorkspace.ID,
		KeySpaceID:     r.RootKeySpace.ID,
		ForWorkspaceID: &r.UserWorkspace.ID,
	})
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+key.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	route := &updatekey.Handler{
		DB:           h.DB,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}
	h.Register(route)
	admin := h.CreateRootKey(r.RootWorkspace.ID, "unkey:v1:"+r.RootWorkspace.ID+":**#*")
	res := testutil.CallRoute[updatekey.Request, updatekey.Response](h, route, http.Header{
		"Authorization": {"Bearer " + admin}, "Content-Type": {"application/json"},
	}, updatekey.Request{KeyId: key.KeyID, Enabled: new(false)})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
}

// TestNewRootKeyAuthenticationChecksLifecycle guarantees the new storage does not
// bypass lifecycle checks. For example, expired or disabled keys fail auth, as do
// keys whose target workspace is missing or disabled.
func TestNewRootKeyAuthenticationChecksLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name      string
		statement string
		target    string
		code      codes.URN
	}{
		{"disabled", "UPDATE unkey_root_keys SET enabled = FALSE WHERE id = ?", "key", codes.Auth.Authorization.KeyDisabled.URN()},
		{"expired", "UPDATE unkey_root_keys SET expires = '2000-01-01' WHERE id = ?", "key", codes.Auth.Authorization.Forbidden.URN()},
		{"deleted", "UPDATE unkey_root_keys SET deleted_at = 1 WHERE id = ?", "key", codes.Auth.Authentication.KeyNotFound.URN()},
		{"missing target", "UPDATE unkey_root_keys SET for_workspace_id = 'ws_missing' WHERE id = ?", "key", codes.Data.Workspace.NotFound.URN()},
		{"disabled target", "UPDATE workspaces SET enabled = FALSE WHERE id = ?", "target", codes.Auth.Authorization.WorkspaceDisabled.URN()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, route, p := newHarness(t)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{}})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			targets := map[string]string{
				"key":    res.Body.Data.KeyId,
				"target": p.AuthorizedWorkspaceID,
			}
			_, err := h.DB.RW().ExecContext(t.Context(), tt.statement, targets[tt.target])
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.Header.Set("Authorization", "Bearer "+res.Body.Data.Key)
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
			for range 2 {
				_, err = h.Keys.GetRootKey(t.Context(), session)
				require.Error(t, err)
				code, ok := fault.GetCode(err)
				require.True(t, ok)
				require.Equal(t, tt.code, code)
			}
		})
	}
}
