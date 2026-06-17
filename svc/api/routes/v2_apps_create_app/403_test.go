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

func TestCreateAppForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			return &ctrlv1.CreateAppResponse{Id: uid.New(uid.AppPrefix)}, nil
		},
	}
	route := &handler.Handler{
		DB:         h.DB,
		Auditlogs:  h.Auditlogs,
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
		{name: "exact project permission", permissions: []string{"unkey:v1:" + workspace.ID + ":projects/" + project.ID + "#create_app"}, shouldPass: true},
		{name: "wildcard project permission", permissions: []string{"unkey:v1:" + workspace.ID + ":projects/*#create_app"}, shouldPass: true},
		{name: "admin permission", permissions: []string{"unkey:v1:" + workspace.ID + ":**#*"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"unkey:v1:" + workspace.ID + ":projects/*#create_project"}, shouldPass: false},
		{name: "legacy dotted does not satisfy urn check", permissions: []string{"project.*.create_app"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
		{name: "wrong workspace", permissions: []string{"unkey:v1:ws_wrong:projects/*#create_app"}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				ProjectSlug: projectSlug,
				Name:        "App",
				Slug:        "app-slug",
			})
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
			}
		})
	}
}
