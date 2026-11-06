package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_reroll_key"
)

func TestRerollRootKeyRejectsMissingCallerWithExistingTarget(t *testing.T) {
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
	before := liveRootKeyIDs(t, h, workspace.ID)
	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers(caller.Key),
		handler.Request{KeyId: target.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, codes.Auth.Authorization.Forbidden.DocsURL(), res.Body.Error.Type)
	require.Equal(t, "The calling root key is no longer available.", res.Body.Error.Detail)
	require.Equal(t, before, liveRootKeyIDs(t, h, workspace.ID))
}

func TestRerollRootKeyCannotWidenNetworks(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#write"},
		IpAllowlist: []string{"192.0.2.0/25"},
	})
	unrestricted := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	res := call(h, route, caller.Key, handler.Request{KeyId: unrestricted.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	res = call(h, route, caller.Key, handler.Request{KeyId: caller.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	session := rootKeySession(t, res.Body.Data.Key)
	session.SetClientIP(netip.MustParseAddr("192.0.2.128"))
	_, err := h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
	session.SetClientIP(netip.MustParseAddr("192.0.2.127"))
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
}

// TestRerollPreservesTargetNetworksAfterDisablement prevents rotation from
// replacing a target's narrow policy with its caller's broader policy.
func TestRerollPreservesTargetNetworksAfterDisablement(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	require.NoError(t, db.Query.UpdateIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateIPAllowlistEnabledParams{
		WorkspaceID: workspace.ID,
		Enabled:     false,
	}))
	caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#write"},
		IpAllowlist: []string{"192.0.2.0/24"},
	})
	target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
		WorkspaceID: workspace.ID,
		IpAllowlist: []string{"192.0.2.0/26"},
	})
	res := call(h, route, caller.Key, handler.Request{KeyId: target.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.0/26"}, []string(stored.IpAllowlist))
	session := rootKeySession(t, res.Body.Data.Key)
	session.SetClientIP(netip.MustParseAddr("192.0.2.63"))
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	session.SetClientIP(netip.MustParseAddr("192.0.2.64"))
	_, err = h.Keys.GetRootKey(t.Context(), session)
	require.Error(t, err)
}

// TestRerollRootKeyCopiesPermissionsAndRevokesOriginal guarantees rotation keeps
// access while expiration 0 revokes the original even when its cache is warm.
func TestRerollRootKeyCopiesPermissionsAndRevokesOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{permission}})
	sourceSession := rootKeySession(t, source.Key)
	_, err := h.Keys.GetRootKey(t.Context(), sourceSession)
	require.NoError(t, err)
	caller := h.CreateRootKey(workspace.ID,
		"unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#write",
		"unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#delete",
		permission,
	)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue[int64](0)})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err = h.Keys.GetRootKey(t.Context(), sourceSession)
	require.Error(t, err)
	rerolled, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, res.Body.Data.Key))
	require.NoError(t, err)
	require.Equal(t, []string{permission}, rerolled.Permissions)
}

// TestRerollRootKeyExpirationRequiresDeletePermission guarantees callers cannot
// shorten or revoke an original root key with write access alone.
func TestRerollRootKeyExpirationRequiresDeletePermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#write")
	before := liveRootKeyIDs(t, h, workspace.ID)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue[int64](0)})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, before, liveRootKeyIDs(t, h, workspace.ID))
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, source.Key))
	require.NoError(t, err)
}

// TestRerollRootKeyRequiresExplicitNullToKeepOriginal guarantees callers must
// choose whether the original survives, and null does not need delete access.
func TestRerollRootKeyRequiresExplicitNullToKeepOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/"+source.KeyID+"#write")
	missing := testutil.CallRoute[map[string]any, handler.Response](h, route, headers(caller), map[string]any{"keyId": source.KeyID})
	require.Equal(t, http.StatusBadRequest, missing.Status, "%s", missing.RawBody)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	_, err := h.Keys.GetRootKey(t.Context(), rootKeySession(t, source.Key))
	require.NoError(t, err)
}

// TestRerollRootKeyRejectsPermissionEscalationWithoutWrites guarantees a caller
// cannot receive a secret for permissions it does not already hold.
func TestRerollRootKeyRejectsPermissionEscalationWithoutWrites(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{"unkey:v1:" + workspace.ID + ":**#*"}})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	before := liveRootKeyIDs(t, h, workspace.ID)
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, before, liveRootKeyIDs(t, h, workspace.ID))
}

// TestRerollRootKeyOverlapCannotExtendOriginal guarantees a grace period never
// extends the original key's existing expiration.
func TestRerollRootKeyOverlapCannotExtendOriginal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	expires := h.Clock.Now().Add(time.Hour).Truncate(time.Millisecond)
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Expires: &expires})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", "unkey:v1:"+workspace.ID+":rootKeys/*#delete")
	res := call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullableWithValue((24 * time.Hour).Milliseconds())})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	original, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), source.KeyID)
	require.NoError(t, err)
	require.Equal(t, expires.UnixMilli(), original.Expires.Int64)
	rerolled, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, expires.UnixMilli(), rerolled.Expires.Int64)
}

// TestRerollRootKeyRejectsLongerLivedSource guarantees an expiring caller cannot
// obtain a longer-lived credential, even when revoking the original immediately.
func TestRerollRootKeyRejectsLongerLivedSource(t *testing.T) {
	for _, expiration := range []string{"unlimited", "later"} {
		for _, overlap := range []string{"keep", "revoke"} {
			t.Run(expiration+"/"+overlap, func(t *testing.T) {
				h := testutil.NewHarness(t)
				route := newRoute(h)
				workspace := h.Resources().UserWorkspace
				callerExpires := h.Clock.Now().Add(time.Hour).Truncate(time.Millisecond)
				var sourceExpires *time.Time
				if expiration == "later" {
					sourceExpires = new(callerExpires.Add(time.Millisecond))
				}
				permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
				source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
					WorkspaceID: workspace.ID,
					Expires:     sourceExpires,
					Permissions: []string{permission},
				})
				caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
					WorkspaceID: workspace.ID,
					Expires:     &callerExpires,
					Permissions: []string{
						"unkey:v1:" + workspace.ID + ":rootKeys/" + source.KeyID + "#write",
						"unkey:v1:" + workspace.ID + ":rootKeys/" + source.KeyID + "#delete",
						permission,
					},
				})
				before := liveRootKeyIDs(t, h, workspace.ID)
				originalBefore, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), source.KeyID)
				require.NoError(t, err)
				grace := nullable.NewNullNullable[int64]()
				if overlap == "revoke" {
					grace = nullable.NewNullableWithValue[int64](0)
				}
				res := call(h, route, caller.Key, handler.Request{KeyId: source.KeyID, Expiration: grace})
				require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
				require.Equal(t, before, liveRootKeyIDs(t, h, workspace.ID))
				originalAfter, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), source.KeyID)
				require.NoError(t, err)
				require.Equal(t, originalBefore, originalAfter)
			})
		}
	}
}

// TestAgentRerollCannotOutliveToken guarantees an agent access token cannot
// reroll a root key that outlives the token, which would hand the agent a new
// secret for a longer-lived key.
func TestAgentRerollCannotOutliveToken(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	expiresAt := h.Clock.Now().Add(time.Hour).Truncate(time.Millisecond)
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	p := &principal.Principal{
		Type:    principal.TypeJWT,
		Subject: principal.Subject{ID: uid.New(uid.TestPrefix), Type: principal.SubjectTypeUser},
		Source: principal.AgentSource{
			RegistrationID: "agent_reg_" + uid.New(uid.TestPrefix),
			ExpiresAt:      expiresAt,
		},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions: []string{
			"unkey:v1:" + workspace.ID + ":rootKeys/*#write",
			"unkey:v1:" + workspace.ID + ":rootKeys/*#delete",
			permission,
		},
	}
	route := &handler.Handler{DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
	h.Register(route, append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			s.SetPrincipal(p)
			return next(ctx, s)
		}
	})...)

	for _, tt := range []struct {
		name    string
		expires *time.Time
		status  int
	}{
		{"unlimited", nil, http.StatusBadRequest},
		{"later", new(expiresAt.Add(time.Millisecond)), http.StatusBadRequest},
		{"equal", &expiresAt, http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			target := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
				WorkspaceID: workspace.ID,
				Expires:     tt.expires,
				Permissions: []string{permission},
			})
			before := liveRootKeyIDs(t, h, workspace.ID)
			res := call(h, route, "test", handler.Request{KeyId: target.KeyID, Expiration: nullable.NewNullNullable[int64]()})
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			if tt.status != http.StatusOK {
				require.Equal(t, before, liveRootKeyIDs(t, h, workspace.ID))
			}
		})
	}
}

// TestRerollRootKeyPreservesSourceExpiration guarantees an expiring caller can
// rotate keys that expire at or before its own expiration without extending them.
func TestRerollRootKeyPreservesSourceExpiration(t *testing.T) {
	for _, remaining := range []time.Duration{0, time.Millisecond} {
		t.Run(remaining.String(), func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := newRoute(h)
			workspace := h.Resources().UserWorkspace
			callerExpires := h.Clock.Now().Add(time.Hour).Truncate(time.Millisecond)
			sourceExpires := callerExpires.Add(-remaining)
			source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Expires: &sourceExpires})
			caller := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{
				WorkspaceID: workspace.ID,
				Expires:     &callerExpires,
				Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/" + source.KeyID + "#write"},
			})
			res := call(h, route, caller.Key, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			rerolled, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
			require.NoError(t, err)
			require.True(t, rerolled.Expires.Valid)
			require.Equal(t, sourceExpires.UnixMilli(), rerolled.Expires.Int64)
			_, err = h.Keys.GetRootKey(t.Context(), rootKeySession(t, res.Body.Data.Key))
			require.NoError(t, err)
		})
	}
}

// TestRerollRootKeyIgnoresLegacyKeys guarantees rootKeys.rerollKey manages only
// keys in the new root-key store.
func TestRerollRootKeyIgnoresLegacyKeys(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	legacy := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: h.Resources().RootWorkspace.ID, KeySpaceID: h.Resources().RootKeySpace.ID, ForWorkspaceID: &workspace.ID})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write")
	res := call(h, route, caller, handler.Request{KeyId: legacy.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
}

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, RootKeyCache: h.Caches.RootKeyByHash, Clock: h.Clock}
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

func headers(bearer string) http.Header {
	return http.Header{"Authorization": {"Bearer " + bearer}, "Content-Type": {"application/json"}}
}

func call(h *testutil.Harness, route *handler.Handler, bearer string, req handler.Request) testutil.TestResponse[handler.Response] {
	return testutil.CallRoute[handler.Request, handler.Response](h, route, headers(bearer), req)
}

func liveRootKeyIDs(t *testing.T, h *testutil.Harness, workspaceID string) []string {
	t.Helper()
	rows, err := db.Query.ListRootKeys(t.Context(), h.DB.RO(), db.ListRootKeysParams{WorkspaceID: workspaceID, IDCursor: "", Limit: 1000})
	require.NoError(t, err)
	return array.Map(rows, func(row db.ListRootKeysRow) string { return row.ID })
}

// TestRerollRootKeyWaitsForOriginalRowLock guarantees rerolls serialize with
// other transactions that hold the original key row lock and copy the permissions
// that remain after the lock is released.
func TestRerollRootKeyWaitsForOriginalRowLock(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	permission := "unkey:v1:" + workspace.ID + ":rootKeys/*#read"
	source := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID, Permissions: []string{permission}})
	caller := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", permission)
	tx, err := h.DB.RW().Begin(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	_, err = db.Query.FindUnkeyRootKeyByIDForUpdate(t.Context(), tx, source.KeyID)
	require.NoError(t, err)

	done := make(chan testutil.TestResponse[handler.Response], 1)
	go func() {
		done <- call(h, route, caller, handler.Request{KeyId: source.KeyID, Expiration: nullable.NewNullNullable[int64]()})
	}()
	select {
	case res := <-done:
		t.Fatalf("reroll completed while original row was locked: %d %s", res.Status, res.RawBody)
	case <-time.After(200 * time.Millisecond):
	}
	require.NoError(t, db.Query.DeleteUnkeyPermissionsByPrincipal(t.Context(), tx, db.DeleteUnkeyPermissionsByPrincipalParams{
		WorkspaceID:   workspace.ID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   source.KeyID,
	}))
	require.NoError(t, tx.Commit())
	select {
	case res := <-done:
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		permissions, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
			WorkspaceID: workspace.ID, PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey, PrincipalID: res.Body.Data.KeyId,
		})
		require.NoError(t, err)
		require.Empty(t, permissions)
	case <-time.After(5 * time.Second):
		t.Fatal("reroll did not complete after row lock was released")
	}
}
