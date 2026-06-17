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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_get_project"
)

func TestGetProjectForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Seed a real project so the lookup succeeds and authorization is the only gate.
	slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        slug,
	})

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"project.*.read_project"}, shouldPass: true},
		{name: "specific permission", permissions: []string{fmt.Sprintf("project.%s.read_project", project.ID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "project.*.read_project"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"project.*.create_project"}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Slug: slug})
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}

			// An unauthorized same-workspace key must not be able to tell a real
			// project apart from a missing one. The response mirrors the 404 a
			// nonexistent slug returns and must never echo the project ID.
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, project.ID, "response must not leak the project ID for %v", tc.permissions)
			require.NotContains(t, res.RawBody, slug, "response must not echo the requested slug for %v", tc.permissions)
		})
	}
}

// TestGetProjectExistenceNotLeaked asserts the existence oracle is closed: an
// unauthorized key querying a real slug and a nonexistent slug receives
// byte-identical responses (modulo the per-request ID), so it cannot enumerate
// the workspace's project inventory.
func TestGetProjectExistenceNotLeaked(t *testing.T) {
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

	missingSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))

	// Key in the same workspace with no project read grant at all.
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Slug: realSlug})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Slug: missingSlug})

	require.Equal(t, http.StatusNotFound, realRes.Status, "real slug should look not-found, got: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "missing slug should be not-found, got: %s", missingRes.RawBody)

	require.NotContains(t, realRes.RawBody, project.ID, "response must not leak the project ID")

	// The only field allowed to differ is the per-request ID in meta.
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail, "error detail must be identical for real and missing slugs")
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type, "error type must be identical for real and missing slugs")
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status, "error status must be identical for real and missing slugs")
}
