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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
)

func TestDeleteAppDeleteProtection(t *testing.T) {
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
		ID:               uid.New(uid.AppPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		Name:             "Protected",
		Slug:             appSlug,
		DefaultBranch:    "main",
		DeleteProtection: true,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, headers, handler.Request{
		AppId: app.ID,
	})
	require.Equal(t, 412, res.Status, "expected 412, received: %s", res.RawBody)
	require.NotNil(t, res.Body.Error)
	require.Equal(t, "This app has delete protection enabled. Disable it before attempting to delete.", res.Body.Error.Detail)

	// The control plane must not be invoked when delete protection blocks the request.
	require.Empty(t, ctrlClient.DeleteAppCalls)
}
