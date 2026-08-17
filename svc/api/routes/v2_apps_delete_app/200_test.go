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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
)

func TestDeleteAppSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	restateClient, deletes := newRecordingRestate(t)

	route := &handler.Handler{
		DB:      h.DB,
		Restate: restateClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.delete_app")
	rootKeyID, err := db.Query.FindKeyIDByHash(ctx, h.DB.RO(), hash.Sha256(rootKey))
	require.NoError(t, err)
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
	require.Equal(t, 202, res.Status, "expected 202, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	observed := testutil.Receive(t, deletes, 10*time.Second)
	require.Equal(t, app.ID, observed.virtualObjectKey)
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
	require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
	require.NotEmpty(t, observed.request.GetCorrelationId())

	// The route only submits the asynchronous workflow.
	_, err = db.Query.FindAppById(ctx, h.DB.RO(), app.ID)
	require.NoError(t, err)
}
