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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_update_project"
)

func TestUpdateProjectForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "exact permission", permissions: []string{"project.*.update_project"}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "project.*.update_project"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"project.*.read_project"}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.create_api"}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
			project := h.CreateProject(seed.CreateProjectRequest{
				ID:          uid.New(uid.ProjectPrefix),
				WorkspaceID: workspace.ID,
				Name:        "Forbidden Test",
				Slug:        slug,
			})

			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			newName := "Updated"
			req := handler.Request{Project: project.ID, Name: &newName}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
			}
		})
	}
}
