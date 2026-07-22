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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_github_connection"
)

func TestCreateGithubConnectionValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.connect_github")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// A real app so the repository rejection is the only thing under test.
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

	badRepoNoSlash := "not-a-repo"
	badRepoWrongHost := "https://gitlab.com/owner/name"
	badRepoSpace := "owner name"

	testCases := []struct {
		name string
		req  handler.Request
	}{
		// Spec-level (openapi validation middleware).
		{name: "missing project", req: handler.Request{App: app.ID}},
		{name: "missing app", req: handler.Request{Project: project.ID}},
		// Handler-level repository rejection.
		{name: "repository without owner", req: handler.Request{Project: project.ID, App: app.ID, Repository: &badRepoNoSlash}},
		{name: "repository wrong host", req: handler.Request{Project: project.ID, App: app.ID, Repository: &badRepoWrongHost}},
		{name: "repository with space", req: handler.Request{Project: project.ID, App: app.ID, Repository: &badRepoSpace}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.req, res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		})
	}
}
