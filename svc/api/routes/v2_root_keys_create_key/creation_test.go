package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	rootkey "github.com/unkeyed/unkey/pkg/auth/root_key"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func newHarness(t *testing.T) (*testutil.Harness, *handler.Handler, *principal.Principal) {
	t.Helper()
	h := testutil.NewHarness(t)
	resources := h.Resources()
	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
		InternalWorkspaceID: resources.RootWorkspace.ID,
		InternalKeyspaceID:  resources.RootKeySpace.ID,
		InternalProjectID:   resources.RootKeySpace.ProjectID,
	}
	p := &principal.Principal{
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: "user_admin", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: resources.UserWorkspace.ID,
		Permissions:           []string{"unkey:v1:" + resources.UserWorkspace.ID + ":**#*"},
	}
	middlewares := append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			s.SetPrincipal(p)
			return next(ctx, s)
		}
	})
	h.Register(route, middlewares...)
	return h, route, p
}

func TestCreatePermissionCountLimits(t *testing.T) {
	h, route, p := newHarness(t)
	for _, tt := range []struct {
		name   string
		count  int
		status int
	}{
		{"empty", 0, http.StatusOK},
		{"maximum", 1000, http.StatusOK},
		{"above maximum", 1001, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requested := make([]string, tt.count)
			for i := range requested {
				requested[i] = p.Permissions[0]
			}
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: requested})
			require.Equal(t, tt.status, res.Status, "%s", res.RawBody)
			if tt.status != http.StatusOK {
				require.Equal(t, before, snapshot(t, h))
				return
			}
			grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
			require.NoError(t, err)
			if tt.count == 0 {
				require.Empty(t, grants)
			} else {
				require.Equal(t, []string{p.Permissions[0]}, grants)
			}
		})
	}
}

func TestCreateRejectsLegacyPermissionsAtomically(t *testing.T) {
	h, route, p := newHarness(t)
	permission := "unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write"
	legacy := "workspace.*.create_root_key"
	p.Permissions = append(p.Permissions, "*", legacy)
	for _, tt := range []struct {
		name      string
		requested []string
	}{
		{"legacy only", []string{legacy}},
		{"URN then legacy", []string{permission, legacy}},
		{"legacy then URN", []string{legacy, permission}},
		{"literal star", []string{"*"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{Permissions: tt.requested})
			require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
			require.Equal(t, before, snapshot(t, h))
		})
	}
}

func TestCreateStoresMaximumDistinctPermissions(t *testing.T) {
	h, route, p := newHarness(t)
	base := "unkey:v1:" + p.AuthorizedWorkspaceID + ":"
	requested := []string{base + "rootKeys/*#write"}
	p.Permissions = []string{requested[0]}
	for i := range 999 {
		project := fmt.Sprintf("%sprojects/proj_%04d", base, i)
		p.Permissions = append(p.Permissions, project+"/keyspaces/*#read")
		requested = append(requested, project+"/keyspaces/ks_one#read")
	}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: requested})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	stored, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.ElementsMatch(t, requested, stored)
	require.Len(t, h.FindAuditLogsByTargetID(t.Context(), t, res.Body.Data.KeyId), 1001)
}

func TestCreateStoresPermissionWithoutLegacyEquivalent(t *testing.T) {
	h, route, p := newHarness(t)
	grant := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/*/apps/*/environments/*/deployments/*#delete"

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{grant}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.Equal(t, []string{grant}, grants)
}

func TestCreateStoresV1SystemKeyAndPermissions(t *testing.T) {
	h, route, p := newHarness(t)
	resources := h.Resources()
	permission := "unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write"
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
		Permissions: []string{permission, permission},
	})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Regexp(t, `^unkey_[1-9A-HJ-NP-Za-km-z]{8}unkeyv1[1-9A-HJ-NP-Za-km-z]{42}$`, res.Body.Data.Key)
	key, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, resources.RootWorkspace.ID, key.WorkspaceID)
	require.Equal(t, resources.RootKeySpace.ID, key.KeyAuthID)
	require.Equal(t, resources.UserWorkspace.ID, key.ForWorkspaceID.String)
	require.False(t, key.IdentityID.Valid)
	require.False(t, key.Expires.Valid)
	grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: key.ID})
	require.NoError(t, err)
	require.Equal(t, []string{permission}, grants)
	logs := h.FindAuditLogsByTargetID(t.Context(), t, key.ID)
	require.Len(t, logs, 2)
	for _, log := range logs {
		require.Equal(t, "user", log.Actor.Type)
		require.Equal(t, p.Subject.ID, log.Actor.ID)
		require.Equal(t, p.AuthorizedWorkspaceID, log.WorkspaceID)
		require.NotEmpty(t, log.CorrelationID)
		require.Equal(t, logs[0].CorrelationID, log.CorrelationID)
	}
	request := httptest.NewRequest(http.MethodPost, route.Path(), nil)
	request.Header.Set("Authorization", "Bearer "+res.Body.Data.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	resolved, err := rootkey.NewResolver(h.Keys).Resolve(t.Context(), session)
	require.NoError(t, err)
	require.Equal(t, p.AuthorizedWorkspaceID, resolved.AuthorizedWorkspaceID)
	require.ElementsMatch(t, grants, resolved.Permissions)

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: resources.UserWorkspace.ID})
	urn := "unkey:v1:" + resources.UserWorkspace.ID + ":projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#decrypt"
	p.Permissions = []string{urn, "unkey:v1:" + resources.UserWorkspace.ID + ":rootKeys/*#write"}
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
		Permissions: []string{urn, urn},
	})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	grants, err = db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.Equal(t, []string{urn}, grants)

	t.Run("expired timestamps are rejected", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
			Permissions: []string{urn}, Expires: nullable.NewNullableWithValue(h.Clock.Now().UnixMilli()),
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
	})
	t.Run("future expiration is stored", func(t *testing.T) {
		expires := h.Clock.Now().UnixMilli() + 60000
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
			Permissions: []string{urn}, Expires: nullable.NewNullableWithValue(expires),
		})
		require.Equal(t, http.StatusOK, res.Status)
		key, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
		require.NoError(t, err)
		require.True(t, key.Expires.Valid)
		require.Equal(t, expires, key.Expires.Time.UnixMilli())
	})
	t.Run("mismatched internal ownership fails closed", func(t *testing.T) {
		route.InternalKeyspaceID = api.KeyAuthID.String
		t.Cleanup(func() { route.InternalKeyspaceID = resources.RootKeySpace.ID })
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
			Permissions: []string{urn},
		})
		require.Equal(t, http.StatusInternalServerError, res.Status)
	})
}
