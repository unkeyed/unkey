package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
)

func TestDeleteAppSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockAppClient{}
	route := &handler.Handler{
		DB:         h.DB,
		CtrlClient: ctrlClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.delete_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        projectSlug,
	})

	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Doomed",
		Slug:          appSlug,
		DefaultBranch: "main",
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	// Deletion is delegated to the control plane, which also writes the audit
	// log. Assert the RPC carried the resolved app id and the actor. The row is
	// torn down asynchronously, so we do not assert it is gone here.
	require.Len(t, ctrlClient.DeleteAppCalls, 1)
	require.Equal(t, app.ID, ctrlClient.DeleteAppCalls[0].GetAppId())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, ctrlClient.DeleteAppCalls[0].GetActor().GetType())

	// Sanity: the app still exists in our DB because the cascade runs in the
	// (mocked) control plane, not in this handler.
	_, err := db.Query.FindAppByWorkspaceAndId(ctx, h.DB.RO(), db.FindAppByWorkspaceAndIdParams{
		WorkspaceID: workspace.ID,
		ID:          app.ID,
	})
	require.NoError(t, err)
}
