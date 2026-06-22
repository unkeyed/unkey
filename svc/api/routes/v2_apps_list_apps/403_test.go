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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_list_apps"
)

func TestListAppsForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Seed a real project with an app so the lookup succeeds and authorization
	// is the only gate.
	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
		DefaultBranch: "main",
	})

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard app permission", permissions: []string{"project.*.read_app"}, shouldPass: true},
		{name: "specific project permission", permissions: []string{fmt.Sprintf("project.%s.read_app", project.ID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "project.*.read_app"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"project.*.create_project"}, shouldPass: false},
		{name: "read does not match create", permissions: []string{"project.*.create_app"}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "urn style does not satisfy legacy check", permissions: []string{"unkey:v1:" + workspace.ID + ":projects/*/apps/*#read_app"}, shouldPass: false},
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
			})
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}

			// An unauthorized same-workspace key must not be able to tell a real
			// project apart from a missing one. The response mirrors the 404 a
			// nonexistent slug returns and must never echo the project or app IDs.
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, project.ID, "response must not leak the project ID for %v", tc.permissions)
			require.NotContains(t, res.RawBody, app.ID, "response must not leak any app ID for %v", tc.permissions)
		})
	}
}

// TestListAppsExistenceNotLeaked asserts the existence oracle is closed: an
// unauthorized key listing a real project and a nonexistent project receives
// byte-identical responses (modulo the per-request ID), so it cannot enumerate
// the workspace's project inventory.
func TestListAppsExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	realSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        realSlug,
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
		DefaultBranch: "main",
	})

	missingSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))

	// Key in the same workspace with an unrelated grant but no read_app action.
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: project.Slug})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: missingSlug})

	require.Equal(t, http.StatusNotFound, realRes.Status, "real project should look not-found, got: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "missing project should be not-found, got: %s", missingRes.RawBody)

	require.NotContains(t, realRes.RawBody, project.ID, "response must not leak the project ID")
	require.NotContains(t, realRes.RawBody, app.ID, "response must not leak any app ID")

	// The only field allowed to differ is the per-request ID in meta.
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail, "error detail must be identical for real and missing projects")
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type, "error type must be identical for real and missing projects")
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status, "error status must be identical for real and missing projects")
}
