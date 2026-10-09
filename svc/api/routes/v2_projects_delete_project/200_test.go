package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_delete_project"
	gethandler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_get_project"
	listhandler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_list_projects"
)

// TestDeleteProjectSuccessfully guarantees that the API submits the resolved
// project key and audit payload through a real Restate server, and that the
// project is hidden from reads as soon as the 202 is returned, before the
// asynchronous teardown removes the row.
func TestDeleteProjectSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	restateClient, deletes := newRecordingRestate(t)

	route := &handler.Handler{
		DB:      h.DB,
		Restate: restateClient,
	}
	h.Register(route)
	listRoute := &listhandler.Handler{DB: h.DB}
	h.Register(listRoute)
	getRoute := &gethandler.Handler{DB: h.DB}
	h.Register(getRoute)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.delete_project", "project.*.read_project")
	rootKeyID, err := db.Query.FindKeyIDByHash(ctx, h.DB.RO(), hash.Sha256(rootKey))
	require.NoError(t, err)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	kept := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "kept-project",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("kept"), "_", "-")),
	})

	// The handler resolves a project by either its id or its slug, so run the
	// same assertions against both identifiers.
	testCases := []struct {
		name       string
		identifier func(db.Project) string
	}{
		{name: "deletes project by id", identifier: func(p db.Project) string { return p.ID }},
		{name: "deletes project by slug", identifier: func(p db.Project) string { return p.Slug }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slug := strings.ToLower(strings.ReplaceAll(uid.New("kebap"), "_", "-"))
			project := h.CreateProject(seed.CreateProjectRequest{
				ID:               uid.New(uid.ProjectPrefix),
				WorkspaceID:      workspace.ID,
				Name:             "kebap-project",
				Slug:             slug,
				DeleteProtection: false,
			})

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project: tc.identifier(project),
			})
			require.Equal(t, 202, res.Status, "expected 202, received: %s", res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)

			listed := testutil.CallRoute[listhandler.Request, listhandler.Response](h, listRoute, headers, listhandler.Request{})
			require.Equal(t, 200, listed.Status, "expected 200, received: %s", listed.RawBody)
			listedIDs := make([]string, 0, len(listed.Body.Data))
			for _, p := range listed.Body.Data {
				listedIDs = append(listedIDs, p.Id)
			}
			require.Contains(t, listedIDs, kept.ID)
			require.NotContains(t, listedIDs, project.ID, "project pending deletion must not be listed")

			got := testutil.CallRoute[gethandler.Request, openapi.NotFoundErrorResponse](h, getRoute, headers, gethandler.Request{
				Project: tc.identifier(project),
			})
			require.Equal(t, http.StatusNotFound, got.Status, "project pending deletion must read as 404, received: %s", got.RawBody)

			observed := testutil.Receive(t, deletes, 10*time.Second)
			require.Equal(t, project.ID, observed.virtualObjectKey)
			require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
			require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
			require.NotEmpty(t, observed.request.GetCorrelationId())
		})
	}
}
