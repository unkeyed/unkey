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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
	gethandler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_get_app"
	listhandler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_list_apps"
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
	listRoute := &listhandler.Handler{DB: h.DB}
	h.Register(listRoute)
	getRoute := &gethandler.Handler{DB: h.DB}
	h.Register(getRoute)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.delete_app", "app.*.read_app")
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
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Doomed",
		Slug:        appSlug,
	})
	kept := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Kept",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("kept"), "_", "-")),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
	})
	require.Equal(t, 202, res.Status, "expected 202, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	listed := testutil.CallRoute[listhandler.Request, listhandler.Response](h, listRoute, headers, listhandler.Request{
		Project: project.ID,
	})
	require.Equal(t, 200, listed.Status, "expected 200, received: %s", listed.RawBody)
	listedIDs := make([]string, 0, len(listed.Body.Data))
	for _, a := range listed.Body.Data {
		listedIDs = append(listedIDs, a.Id)
	}
	require.Contains(t, listedIDs, kept.ID)
	require.NotContains(t, listedIDs, app.ID, "app pending deletion must not be listed")

	got := testutil.CallRoute[gethandler.Request, openapi.NotFoundErrorResponse](h, getRoute, headers, gethandler.Request{
		Project: project.ID,
		App:     app.ID,
	})
	require.Equal(t, http.StatusNotFound, got.Status, "app pending deletion must read as 404, received: %s", got.RawBody)

	observed := testutil.Receive(t, deletes, 10*time.Second)
	require.Equal(t, app.ID, observed.virtualObjectKey)
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
	require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
	require.NotEmpty(t, observed.request.GetCorrelationId())
}
