package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_get_environment"
)

func TestGetEnvironmentForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Seed a real environment so the lookup succeeds and authorization is the only gate.
	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})
	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          appSlug,
		DefaultBranch: "main",
	})
	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard environment permission", permissions: []string{"project.*.read_environment"}, shouldPass: true},
		{name: "specific project permission", permissions: []string{fmt.Sprintf("project.%s.read_environment", project.ID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "project.*.read_environment"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"project.*.create_project"}, shouldPass: false},
		{name: "read_app does not match read_environment", permissions: []string{"project.*.read_app"}, shouldPass: false},
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
				Project:     project.ID,
				App:         app.ID,
				Environment: environment.ID,
			})
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}

			// An unauthorized same-workspace key must not be able to tell a real
			// environment apart from a missing one. The response mirrors the 404 a
			// nonexistent id returns and must never echo the environment ID.
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, environment.ID, "response must not leak the environment ID for %v", tc.permissions)
		})
	}
}

// TestGetEnvironmentExistenceNotLeaked asserts the existence oracle is closed: an
// unauthorized key querying a real environment and a nonexistent environment
// receives byte-identical responses (modulo the per-request ID), so it cannot
// enumerate the workspace's environment inventory.
func TestGetEnvironmentExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})
	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          appSlug,
		DefaultBranch: "main",
	})
	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	missingID := uid.New(uid.EnvironmentPrefix)

	// Key in the same workspace with an unrelated grant but no read_environment action.
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: project.ID, App: app.ID, Environment: environment.ID})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: project.ID, App: app.ID, Environment: missingID})

	require.Equal(t, http.StatusNotFound, realRes.Status, "real environment should look not-found, got: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "missing environment should be not-found, got: %s", missingRes.RawBody)

	require.NotContains(t, realRes.RawBody, environment.ID, "response must not leak the environment ID")

	// The only field allowed to differ is the per-request ID in meta.
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail, "error detail must be identical for real and missing environments")
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type, "error type must be identical for real and missing environments")
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status, "error status must be identical for real and missing environments")
}
