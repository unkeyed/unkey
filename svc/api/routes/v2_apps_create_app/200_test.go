package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_app"
)

func TestCreateAppSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	appID := uid.New(uid.AppPrefix)
	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			return &ctrlv1.CreateAppResponse{Id: appID}, nil
		},
	}
	route := &handler.Handler{
		DB:         h.DB,
		Auditlogs:  h.Auditlogs,
		CtrlClient: ctrlClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.create_app")
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

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		ProjectSlug: projectSlug,
		Name:        "Payments API",
		Slug:        "payments-api",
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)
	require.Equal(t, appID, res.Body.Data.AppId)
	require.True(t, strings.HasPrefix(res.Body.Data.AppId, "app_"))

	require.Len(t, ctrlClient.CreateAppCalls, 1)
	call := ctrlClient.CreateAppCalls[0]
	require.Equal(t, workspace.ID, call.GetWorkspaceId())
	require.Equal(t, project.ID, call.GetProjectId())
	require.Equal(t, "Payments API", call.GetName())
	require.Equal(t, "payments-api", call.GetSlug())

	auditLogs := h.FindAuditLogsByTargetID(ctx, t, appID)
	var found bool
	for _, ev := range auditLogs {
		if ev.Event == "app.create" {
			found = true
			require.Equal(t, workspace.ID, ev.WorkspaceID)
			break
		}
	}
	require.True(t, found, "should find an app.create audit log event")
}
