package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environments"
)

func TestListEnvironmentsForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Seed a real project, app, and environment so the lookup succeeds and
	// authorization is the only gate.
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        slug(t),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          slug(t),
		DefaultBranch: "main",
	})
	env := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        slug(t),
	})

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard environment permission", permissions: []string{"environment.*.read_environment"}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "environment.*.read_environment"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"environment.*.read_app"}, shouldPass: false},
		{name: "read_app does not match read_environment", permissions: []string{"environment.*.read_app"}, shouldPass: false},
		{name: "specific environment scope does not match list", permissions: []string{fmt.Sprintf("environment.%s.read_environment", env.ID)}, shouldPass: false},
		{name: "project scope does not match", permissions: []string{fmt.Sprintf("project.%s.read_environment", project.ID)}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project: project.Slug,
				App:     app.Slug,
			})
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}

			// An unauthorized same-workspace key must not be able to tell a real
			// app apart from a missing one. The response mirrors the 404 a
			// nonexistent slug returns and must never echo internal IDs.
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, project.ID, "response must not leak the project ID for %v", tc.permissions)
			require.NotContains(t, res.RawBody, app.ID, "response must not leak the app ID for %v", tc.permissions)
			require.NotContains(t, res.RawBody, env.ID, "response must not leak any environment ID for %v", tc.permissions)
		})
	}
}

// TestListEnvironmentsExistenceNotLeaked asserts the existence oracle is closed:
// an unauthorized key listing a real app and a nonexistent app receives
// byte-identical responses (modulo the per-request ID), so it cannot enumerate
// the workspace's app inventory.
func TestListEnvironmentsExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        slug(t),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          slug(t),
		DefaultBranch: "main",
	})
	env := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        slug(t),
	})

	missingApp := slug(t)

	// Key in the same workspace with an unrelated grant but no read_environment action.
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: project.Slug, App: app.Slug})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: project.Slug, App: missingApp})

	require.Equal(t, http.StatusNotFound, realRes.Status, "real app should look not-found, got: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "missing app should be not-found, got: %s", missingRes.RawBody)

	require.NotContains(t, realRes.RawBody, project.ID, "response must not leak the project ID")
	require.NotContains(t, realRes.RawBody, app.ID, "response must not leak the app ID")
	require.NotContains(t, realRes.RawBody, env.ID, "response must not leak any environment ID")

	// The only field allowed to differ is the per-request ID in meta.
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail, "error detail must be identical for real and missing apps")
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type, "error type must be identical for real and missing apps")
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status, "error status must be identical for real and missing apps")
}
