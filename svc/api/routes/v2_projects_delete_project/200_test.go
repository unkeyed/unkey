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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_delete_project"
)

// TestDeleteProjectSuccessfully guarantees that the API submits the resolved
// project key and audit payload through a real Restate server.
func TestDeleteProjectSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	restateClient, deletes := newRecordingRestate(t)

	route := &handler.Handler{
		DB:      h.DB,
		Restate: restateClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.delete_project")
	rootKeyID, err := db.Query.FindKeyIDByHash(ctx, h.DB.RO(), hash.Sha256(rootKey))
	require.NoError(t, err)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

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

			observed := testutil.Receive(t, deletes, 10*time.Second)
			require.Equal(t, project.ID, observed.virtualObjectKey)
			require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
			require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
			require.NotEmpty(t, observed.request.GetCorrelationId())

			_, err := db.Query.FindProjectByIdOrSlug(ctx, h.DB.RO(), db.FindProjectByIdOrSlugParams{
				WorkspaceID: workspace.ID,
				Project:     tc.identifier(project),
			})
			require.NoError(t, err)
		})
	}
}
