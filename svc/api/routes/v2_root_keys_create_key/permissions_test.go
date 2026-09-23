package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func TestCreateStoresEveryCanonicalResourceAction(t *testing.T) {
	h, route, p := newHarness(t)
	base := "unkey:v1:" + p.AuthorizedWorkspaceID + ":"
	catalog := map[string][]string{
		"github/apps/*":                    {"read", "write", "delete"},
		"rootKeys/*":                       {"write"},
		"projects/*":                       {"read", "write", "delete"},
		"projects/*/apps/*":                {"read", "write", "delete"},
		"projects/*/apps/*/environments/*": {"read", "write", "delete"},
		"projects/*/apps/*/environments/*/deployments/*":      {"read", "write", "delete"},
		"projects/*/apps/*/environments/*/deployments/*/logs": {"read"},
		"projects/*/apps/*/environments/*/domains/*":          {"read", "write", "delete"},
		"projects/*/apps/*/environments/*/variables/*":        {"read", "write", "delete"},
		"projects/*/apps/*/environments/*/gateway/logs":       {"read"},
		"projects/*/apps/*/environments/*/gateway/policies/*": {"read", "write", "delete"},
		"projects/*/identities/*":                             {"read", "write", "delete"},
		"projects/*/keyspaces/*":                              {"read", "write", "delete"},
		"projects/*/keyspaces/*/logs":                         {"read"},
		"projects/*/keyspaces/*/keys/*":                       {"read", "write", "delete", "decrypt", "verify"},
		"projects/*/ratelimits/namespaces/*":                  {"read", "write", "delete", "limit"},
		"projects/*/ratelimits/namespaces/*/logs":             {"read"},
		"projects/*/ratelimits/namespaces/*/overrides/*":      {"read", "write", "delete"},
		"projects/*/rbac/roles/*":                             {"read", "write", "delete"},
		"projects/*/rbac/permissions/*":                       {"read", "write", "delete"},
	}
	requested := []string{
		base + "**#*",
		base + "**#write",
		base + "projects/proj_one/**#decrypt",
		base + "projects/proj_one/apps/logs#write",
		base + "projects/proj_one/ratelimits/namespaces/overrides#limit",
		base + "projects/logs/**#decrypt",
		base + "projects/proj_one/keyspaces/logs/**#decrypt",
		base + "projects/proj_one/ratelimits/namespaces/logs/**#limit",
	}
	for resource, actions := range catalog {
		for _, action := range actions {
			requested = append(requested, base+resource+"#"+action)
		}
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: requested})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.ElementsMatch(t, requested, grants)
}

func TestCreateRejectsInvalidCanonicalResourceActionsAtomically(t *testing.T) {
	h, route, p := newHarness(t)
	base := "unkey:v1:" + p.AuthorizedWorkspaceID + ":"
	for _, permission := range []string{
		base + "rootKeys/*#read",
		base + "projects/*/keyspaces/*/logs#decrypt",
		base + "projects/*/ratelimits/namespaces/*/overrides/*#limit",
		base + "projects/*/apps/*/environments/*/gateway#write",
		base + "projects/*#*",
		base + "projects/*#unknown",
		base + "projects/keys/apps/app_one#decrypt",
		base + "projects/proj_one/keyspaces/keys#decrypt",
		base + "projects/proj_one/apps/keys/**#decrypt",
		base + "projects/*/portals/*/sessions/*#read",
	} {
		t.Run(permission, func(t *testing.T) {
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{base + "projects/*#read", permission}})
			require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
			require.Equal(t, before, snapshot(t, h))
		})
	}
}

func TestCreateTranslatesEveryLegacyResourceFamily(t *testing.T) {
	h, route, p := newHarness(t)
	workspaceID := p.AuthorizedWorkspaceID
	project := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "project", Slug: uid.New("slug"),
	})
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID, ProjectID: project.ID})
	app := h.CreateApp(seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "app", Slug: uid.New("slug"),
	})
	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID, Slug: uid.New("slug"),
	})
	identity := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: uid.New("external")})
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	require.NoError(t, db.Query.InsertRatelimitNamespace(t.Context(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID: namespaceID, Name: namespaceID, WorkspaceID: workspaceID, ProjectID: project.ID,
	}))

	base := "unkey:v1:" + workspaceID + ":"
	legacyToCanonical := map[string]string{
		"workspace.*.install_github":                            base + "github/apps/*#write",
		"workspace.*.create_root_key":                           base + "rootKeys/*#write",
		"api." + api.ID + ".create_key":                         base + "projects/" + project.ID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#write",
		"api." + api.ID + ".update_key":                         base + "projects/" + project.ID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#write",
		"api." + api.ID + ".read_analytics":                     base + "projects/" + project.ID + "/keyspaces/" + api.KeyAuthID.String + "/logs#read",
		"ratelimit." + namespaceID + ".create_namespace":        base + "projects/" + project.ID + "/ratelimits/namespaces/" + namespaceID + "#write",
		"ratelimit." + namespaceID + ".update_namespace":        base + "projects/" + project.ID + "/ratelimits/namespaces/" + namespaceID + "#write",
		"ratelimit." + namespaceID + ".read_analytics":          base + "projects/" + project.ID + "/ratelimits/namespaces/" + namespaceID + "/logs#read",
		"project." + project.ID + ".create_project":             base + "projects/" + project.ID + "#write",
		"project." + project.ID + ".update_project":             base + "projects/" + project.ID + "#write",
		"app." + app.ID + ".update_app":                         base + "projects/" + project.ID + "/apps/" + app.ID + "#write",
		"environment." + environment.ID + ".update_environment": base + "projects/" + project.ID + "/apps/" + app.ID + "/environments/" + environment.ID + "#write",
		"environment." + environment.ID + ".create_deployment":  base + "projects/" + project.ID + "/apps/" + app.ID + "/environments/" + environment.ID + "/deployments/*#write",
		"identity." + identity.ID + ".create_identity":          base + "projects/" + identity.ProjectID + "/identities/" + identity.ID + "#write",
		"identity." + identity.ID + ".update_identity":          base + "projects/" + identity.ProjectID + "/identities/" + identity.ID + "#write",
		"rbac.*.create_role":                                    base + "projects/*/rbac/roles/*#write",
		"rbac.*.read_permission":                                base + "projects/*/rbac/permissions/*#read",
	}
	requested := make([]string, 0, len(legacyToCanonical))
	want := make(map[string]struct{}, len(legacyToCanonical)*2)
	for legacy, canonical := range legacyToCanonical {
		requested = append(requested, legacy)
		want[legacy] = struct{}{}
		want[canonical] = struct{}{}
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: requested})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	grants, err := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
	require.NoError(t, err)
	require.ElementsMatch(t, mapKeys(want), grants)
}

func TestCreateNormalizesLegacyCallerCreateAndUpdateGrantsToWrite(t *testing.T) {
	h, route, p := newHarness(t)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: p.AuthorizedWorkspaceID})
	canonical := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#write"
	for _, legacy := range []string{"api." + api.ID + ".create_key", "api." + api.ID + ".update_key"} {
		t.Run(legacy, func(t *testing.T) {
			p.Permissions = []string{
				"unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write",
				legacy,
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{canonical}})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
		})
	}
}

func TestCreateUsesKeyspaceProjectForRequestedAndCallerLegacyAPIGrants(t *testing.T) {
	h, route, p := newHarness(t)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: p.AuthorizedWorkspaceID})
	keyspaceProject := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: p.AuthorizedWorkspaceID, Name: "keyspace project", Slug: uid.New("slug"),
	})
	_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE key_auth SET project_id = ? WHERE id = ?", keyspaceProject.ID, api.KeyAuthID.String)
	require.NoError(t, err)

	legacy := "api." + api.ID + ".decrypt_key"
	canonical := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + keyspaceProject.ID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#decrypt"

	t.Run("requested legacy grant", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
			"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
		}, handler.Request{Permissions: []string{legacy}})
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

		grants, listErr := db.Query.ListPermissionsByKeyID(t.Context(), h.DB.RO(), db.ListPermissionsByKeyIDParams{KeyID: res.Body.Data.KeyId})
		require.NoError(t, listErr)
		require.ElementsMatch(t, []string{legacy, canonical}, grants)
	})

	t.Run("caller legacy grant", func(t *testing.T) {
		p.Permissions = []string{
			"unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write",
			legacy,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
			"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
		}, handler.Request{Permissions: []string{canonical}})
		require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	})
}

func TestCreateRejectsForeignDeletedAndUnsupportedLegacyResourcesAtomically(t *testing.T) {
	h, route, p := newHarness(t)
	workspaceID := p.AuthorizedWorkspaceID
	foreignWorkspaceID := h.CreateWorkspace().ID
	foreignProject := h.CreateProject(seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: foreignWorkspaceID, Name: "foreign", Slug: uid.New("slug"),
	})
	foreignApp := h.CreateApp(seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: foreignWorkspaceID, ProjectID: foreignProject.ID, Name: "foreign", Slug: uid.New("slug"),
	})
	foreignEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: foreignWorkspaceID, ProjectID: foreignProject.ID, AppID: foreignApp.ID, Slug: uid.New("slug"),
	})
	foreignIdentity := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: foreignWorkspaceID, ExternalID: uid.New("external")})

	deletedAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	deletedKeyspaceAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	deletedNamespaceID := uid.New(uid.RatelimitNamespacePrefix)
	require.NoError(t, db.Query.InsertRatelimitNamespace(t.Context(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID: deletedNamespaceID, Name: deletedNamespaceID, WorkspaceID: workspaceID, ProjectID: deletedAPI.ProjectID,
	}))
	deletedIdentity := h.CreateIdentity(seed.CreateIdentityRequest{WorkspaceID: workspaceID, ExternalID: uid.New("external")})
	_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE apis SET deleted_at_m = 1 WHERE id = ?", deletedAPI.ID)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "UPDATE key_auth SET deleted_at_m = 1 WHERE id = ?", deletedKeyspaceAPI.KeyAuthID.String)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "UPDATE ratelimit_namespaces SET deleted_at_m = 1 WHERE id = ?", deletedNamespaceID)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "UPDATE identities SET deleted = 1 WHERE id = ?", deletedIdentity.ID)
	require.NoError(t, err)

	for _, permission := range []string{
		"project." + foreignProject.ID + ".read_project",
		"app." + foreignApp.ID + ".read_app",
		"environment." + foreignEnvironment.ID + ".read_environment",
		"identity." + foreignIdentity.ID + ".read_identity",
		"api." + deletedAPI.ID + ".read_api",
		"api." + deletedKeyspaceAPI.ID + ".read_api",
		"ratelimit." + deletedNamespaceID + ".read_namespace",
		"identity." + deletedIdentity.ID + ".read_identity",
		"portal.*.create_portal",
		"rbac.concrete.read_role",
		"workspace.concrete.create_root_key",
	} {
		t.Run(permission, func(t *testing.T) {
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{"workspace.*.create_root_key", permission}})
			require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
			require.Equal(t, before, snapshot(t, h))
		})
	}
}

func TestCreateIgnoresForeignLegacyCallerGrant(t *testing.T) {
	h, route, p := newHarness(t)
	foreignAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.CreateWorkspace().ID})
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: p.AuthorizedWorkspaceID})
	p.Permissions = []string{
		"unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#write",
		"api." + foreignAPI.ID + ".decrypt_key",
	}
	requested := "unkey:v1:" + p.AuthorizedWorkspaceID + ":projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#decrypt"

	before := snapshot(t, h)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{requested}})
	require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	require.Equal(t, before, snapshot(t, h))
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
