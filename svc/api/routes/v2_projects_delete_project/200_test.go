package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_delete_project"
)

func TestDeleteProjectSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockProjectClient{}
	route := &handler.Handler{
		DB:         h.DB,
		Auditlogs:  h.Auditlogs,
		CtrlClient: ctrlClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.delete_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Doomed",
		Slug:        slug,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug: slug,
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	// Deletion is delegated to the control plane: assert the RPC was made with
	// the resolved project id. The row is torn down asynchronously, so we do
	// not assert it is gone here.
	require.Len(t, ctrlClient.DeleteProjectCalls, 1)
	require.Equal(t, project.ID, ctrlClient.DeleteProjectCalls[0].GetProjectId())

	auditLogs := h.FindAuditLogsByTargetID(ctx, t, project.ID)
	var found bool
	for _, ev := range auditLogs {
		if ev.Event == "project.delete" {
			found = true
			require.Equal(t, workspace.ID, ev.WorkspaceID)
			break
		}
	}
	require.True(t, found, "should find a project.delete audit log event")

	// Sanity: the project still exists in our DB because the cascade runs in
	// the (mocked) control plane, not in this handler.
	_, err := db.Query.FindProjectByWorkspaceAndSlug(ctx, h.DB.RO(), db.FindProjectByWorkspaceAndSlugParams{
		WorkspaceID: workspace.ID,
		Slug:        slug,
	})
	require.NoError(t, err)
}
