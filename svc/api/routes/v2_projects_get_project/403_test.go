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
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
			}
		})
	}
}
