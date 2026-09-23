package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	rootkey "github.com/unkeyed/unkey/pkg/auth/root_key"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
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

func TestCreateStoresLegacyOnlyWhenRequested(t *testing.T) {
	h, route, p := newHarness(t)
	canonical := "unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write"
	legacy := "workspace.*.create_root_key"
	for _, tt := range []struct {
		name      string
		requested []string
		want      []string
	}{
		{"URN only", []string{canonical, canonical}, []string{canonical}},
		{"legacy only", []string{legacy}, []string{legacy, canonical}},
		{"URN then legacy", []string{canonical, legacy}, []string{legacy, canonical}},
		{"legacy then URN", []string{legacy, canonical}, []string{legacy, canonical}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{Permissions: tt.requested})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, grants)
		})
	}
}

func TestCreateStoresCanonicalGrantWithoutLegacyEquivalent(t *testing.T) {
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

func TestCreateTranslatesProjectLegacyGrant(t *testing.T) {
	h, route, p := newHarness(t)
	project := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: p.AuthorizedWorkspaceID, Name: "project", Slug: uid.New("slug"),
	})
	legacy := "project." + project.ID + ".update_project"
	canonical := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + project.ID + "#write"

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{legacy}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{legacy, canonical}, grants)
}

func TestCreateStoresSystemKeyAndEquivalentGrants(t *testing.T) {
	h, route, p := newHarness(t)
	resources := h.Resources()
	canonical := "unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write"
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
		Permissions: []string{"workspace.*.create_root_key", canonical, "workspace.*.create_root_key"},
	})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.Regexp(t, `^unkey_[1-9A-HJ-NP-Za-km-z]{16,22}$`, res.Body.Data.Key)
	key, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, resources.RootWorkspace.ID, key.WorkspaceID)
	require.Equal(t, resources.RootKeySpace.ID, key.KeyAuthID)
	require.Equal(t, resources.UserWorkspace.ID, key.ForWorkspaceID.String)
	require.False(t, key.IdentityID.Valid)
	require.False(t, key.Expires.Valid)
	grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: key.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"workspace.*.create_root_key", canonical}, grants)
	logs := h.FindAuditLogsByTargetID(t.Context(), t, key.ID)
	require.Len(t, logs, 3)
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
	legacy := "api." + api.ID + ".decrypt_key"
	urn := "unkey:v1:" + resources.UserWorkspace.ID + ":projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#decrypt"
	p.Permissions = []string{urn, "unkey:v1:" + resources.UserWorkspace.ID + ":rootKeys/*#write"}
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
		Permissions: []string{legacy, urn},
	})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	grants, err = db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{legacy, urn}, grants)

	t.Run("expired timestamps are rejected", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
			Permissions: []string{legacy}, Expires: nullable.NewNullableWithValue(h.Clock.Now().UnixMilli()),
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
	})
	t.Run("future expiration is stored", func(t *testing.T) {
		expires := h.Clock.Now().UnixMilli() + 60000
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
			Permissions: []string{legacy}, Expires: nullable.NewNullableWithValue(expires),
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
			Permissions: []string{legacy},
		})
		require.Equal(t, http.StatusInternalServerError, res.Status)
	})
	for legacyAction, action := range map[string]string{"delete_key": "delete", "verify_key": "verify"} {
		t.Run(legacyAction, func(t *testing.T) {
			grant := "unkey:v1:" + resources.UserWorkspace.ID + ":projects/*/keyspaces/*/keys/*#" + action
			p.Permissions = []string{grant, "unkey:v1:" + resources.UserWorkspace.ID + ":rootKeys/*#write"}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
				Permissions: []string{"api.*." + legacyAction},
			})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
			require.NoError(t, err)
			require.ElementsMatch(t, []string{"api.*." + legacyAction, grant}, grants)
		})
	}
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	require.NoError(t, db.Query.InsertRatelimitNamespace(t.Context(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID: namespaceID, Name: namespaceID, WorkspaceID: resources.UserWorkspace.ID, ProjectID: api.ProjectID,
	}))
	for legacyAction, suffix := range map[string]string{
		"limit": "#limit", "read_override": "/overrides/*#read", "set_override": "/overrides/*#write", "delete_override": "/overrides/*#delete",
	} {
		t.Run(legacyAction, func(t *testing.T) {
			grant := "unkey:v1:" + resources.UserWorkspace.ID + ":projects/" + api.ProjectID + "/ratelimits/namespaces/" + namespaceID + suffix
			legacy := "ratelimit." + namespaceID + "." + legacyAction
			p.Permissions = []string{grant, "unkey:v1:" + resources.UserWorkspace.ID + ":rootKeys/*#write"}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{"Authorization": {"Bearer test"}, "Content-Type": {"application/json"}}, handler.Request{
				Permissions: []string{legacy, grant},
			})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
			require.NoError(t, err)
			require.ElementsMatch(t, []string{legacy, grant}, grants)
		})
	}
}
