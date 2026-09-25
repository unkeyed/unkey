package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func TestCreateStoresEveryResourceAction(t *testing.T) {
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
		"projects/*/portals/*/sessions/*":                     {"write"},
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

func TestCreateRejectsInvalidResourceActionsAtomically(t *testing.T) {
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

func TestCreateIgnoresLegacyCallerPermissions(t *testing.T) {
	h, route, p := newHarness(t)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: p.AuthorizedWorkspaceID})
	base := "unkey:v1:" + p.AuthorizedWorkspaceID + ":"
	requested := base + "projects/" + api.ProjectID + "/keyspaces/" + api.KeyAuthID.String + "/keys/*#write"
	for _, legacy := range []string{"*", "api.*.create_key", "api." + api.ID + ".create_key", "api." + api.ID + ".update_key"} {
		t.Run(legacy, func(t *testing.T) {
			p.Permissions = []string{base + "rootKeys/*#write", legacy}
			before := snapshot(t, h)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{requested}})
			require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
			require.Equal(t, before, snapshot(t, h))
		})
	}
}
