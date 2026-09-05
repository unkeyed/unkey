package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
)

func TestDeleteAppRestateFailure(t *testing.T) {
	t.Run("submission rejected", func(t *testing.T) {
		assertRestateFailure(t, testutil.NewRestateIngressClient(t, http.StatusInternalServerError))
	})

	t.Run("transport unavailable", func(t *testing.T) {
		assertRestateFailure(t, testutil.NewUnavailableRestateIngressClient(t))
	})
}

func assertRestateFailure(t *testing.T, restate *restateingress.Client) {
	t.Helper()

	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: restate}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Restate Failure",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("project"), "_", "-")),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Restate Failure",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("app"), "_", "-")),
	})
	rootKey := h.CreateRootKey(workspace.ID, "app.*.delete_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
	})
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to delete app.", res.Body.Error.Detail)
}
