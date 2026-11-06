package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_app"
)

func TestCreateAppForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			return &ctrlv1.CreateAppResponse{Id: uid.New(uid.AppPrefix)}, nil
		},
	}
	route := &handler.Handler{
		DB:         h.DB,
		CtrlClient: ctrlClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Perm",
		Slug:        projectSlug,
	})

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard app permission", permissions: []string{rbac.U(urn.New().Workspace(workspace.ID).Project("*").App("*"), permissions.Write).Value}, shouldPass: true},
		{name: "specific project permission", permissions: []string{rbac.U(urn.New().Workspace(workspace.ID).Project(project.ID).App("*"), permissions.Write).Value}, shouldPass: true},
		{name: "read does not grant create", permissions: []string{rbac.U(urn.New().Workspace(workspace.ID).Project("*").App("*"), permissions.Read).Value}, shouldPass: false},
		{name: "other project", permissions: []string{rbac.U(urn.New().Workspace(workspace.ID).Project(uid.New(uid.ProjectPrefix)).App("*"), permissions.Write).Value}, shouldPass: false},
		{name: "project resource does not grant app create", permissions: []string{rbac.U(urn.New().Workspace(workspace.ID).Project("*"), permissions.Write).Value}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project: projectSlug,
				Name:    "App",
				Slug:    "app-slug",
				Oci:     &openapi.AppOCI{Image: "nginx:stable"},
			})
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
			}
		})
	}
}
