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
	github "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_app"
)

func slug() string {
	return strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
}

func TestCreateAppConnectRepository(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	appID := uid.New(uid.AppPrefix)
	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			return &ctrlv1.CreateAppResponse{Id: appID}, nil
		},
	}
	route := &handler.Handler{
		DB:            h.DB,
		CtrlClient:    ctrlClient,
		Auditlogs:     h.Auditlogs,
		GitHubAppName: "unkey-app",
		GitHubClient: testutil.FakeGitHub{
			Noop:       github.NewNoop(),
			Repo:       github.RepoInfo{ID: 42, FullName: "unkeyed/unkey", DefaultBranch: "main"},
			Accessible: true,
		},
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.create_app", "app.*.connect_repository")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        slug(),
	})
	h.SeedGitHubInstallation(t, workspace.ID, 12345)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		Name:    "Payments API",
		Slug:    slug(),
		Git:     &openapi.AppGitCreateInput{Repository: ptr.P("unkeyed/unkey")},
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, appID, res.Body.Data.AppId)
	require.Len(t, ctrlClient.CreateAppCalls, 1)
	require.NotNil(t, ctrlClient.CreateAppCalls[0].GetGit())

	conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), appID)
	require.NoError(t, err)
	require.Equal(t, "unkeyed/unkey", conn.RepositoryFullName)
	require.Equal(t, int64(42), conn.RepositoryID)
	require.Equal(t, int64(12345), conn.InstallationID)
	require.Equal(t, "main", conn.DefaultBranch.String)

	logs := h.FindAuditLogsByTargetID(ctx, t, appID)
	var found bool
	for _, ev := range logs {
		if ev.Event == "app.connect_repository" {
			found = true
			break
		}
	}
	require.True(t, found, "should find an app.connect_repository audit log event")
}

// TestCreateAppConnectRepositoryWithURNPermission guarantees that one
// app-collection write permission authorizes creation and persists the connection.
func TestCreateAppConnectRepositoryWithURNPermission(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	appID := uid.New(uid.AppPrefix)
	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			return &ctrlv1.CreateAppResponse{Id: appID}, nil
		},
	}
	route := &handler.Handler{
		DB:            h.DB,
		CtrlClient:    ctrlClient,
		Auditlogs:     h.Auditlogs,
		GitHubAppName: "unkey-app",
		GitHubClient: testutil.FakeGitHub{
			Noop:       github.NewNoop(),
			Repo:       github.RepoInfo{ID: 42, FullName: "unkeyed/unkey", DefaultBranch: "main"},
			Accessible: true,
		},
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        slug(),
	})
	permission := fmt.Sprintf("%s#%s", urn.New().Workspace(workspace.ID).Project(project.ID).App("*"), permissions.Write)
	rootKey := h.CreateRootKey(workspace.ID, permission)
	h.SeedGitHubInstallation(t, workspace.ID, 12345)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{
		Project: project.ID,
		Name:    "Payments API",
		Slug:    slug(),
		Git:     &openapi.AppGitCreateInput{Repository: ptr.P("unkeyed/unkey")},
	})

	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, appID, res.Body.Data.AppId)
	require.Len(t, ctrlClient.CreateAppCalls, 1)
	conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), appID)
	require.NoError(t, err)
	require.Equal(t, "unkeyed/unkey", conn.RepositoryFullName)
}

// TestCreateAppConnectRepositoryRejectsURNPermissionOutsideScope
// guarantees that repository connection cannot cross project, workspace, or
// action boundaries after base app creation authorization succeeds.
func TestCreateAppConnectRepositoryRejectsURNPermissionOutsideScope(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			t.Fatal("ctrl CreateApp must not be called with an out-of-scope repository permission")
			return nil, nil
		},
	}
	route := &handler.Handler{
		DB:            h.DB,
		CtrlClient:    ctrlClient,
		Auditlogs:     h.Auditlogs,
		GitHubAppName: "unkey-app",
		GitHubClient:  github.NewNoop(),
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        slug(),
	})
	testCases := []struct {
		name       string
		permission string
	}{
		{
			name:       "wrong project",
			permission: fmt.Sprintf("%s#%s", urn.New().Workspace(workspace.ID).Project(uid.New(uid.ProjectPrefix)).App("*"), permissions.Write),
		},
		{
			name:       "wrong workspace",
			permission: fmt.Sprintf("%s#%s", urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(project.ID).App("*"), permissions.Write),
		},
		{
			name:       "wrong action",
			permission: fmt.Sprintf("%s#%s", urn.New().Workspace(workspace.ID).Project(project.ID).App("*"), permissions.Read),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, "project.*.create_app", testCase.permission)
			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}, handler.Request{
				Project: project.ID,
				Name:    "Payments API",
				Slug:    slug(),
				Git:     &openapi.AppGitCreateInput{Repository: ptr.P("unkeyed/unkey")},
			})

			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
		})
	}
}

func TestCreateGitAppWithoutRepository(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	appID := uid.New(uid.AppPrefix)
	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			return &ctrlv1.CreateAppResponse{Id: appID}, nil
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.create_app")
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        slug(),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{
		Project: project.ID,
		Name:    "Payments API",
		Slug:    slug(),
		Git:     &openapi.AppGitCreateInput{},
	})

	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, ctrlClient.CreateAppCalls, 1)
	require.NotNil(t, ctrlClient.CreateAppCalls[0].GetGit())
	_, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), appID)
	require.True(t, db.IsNotFound(err))
}

func TestCreateAppConnectRepositoryForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockAppClient{
		CreateAppFunc: func(_ context.Context, _ *ctrlv1.CreateAppRequest) (*ctrlv1.CreateAppResponse, error) {
			t.Fatal("ctrl CreateApp must not be called when connect_repository is missing")
			return nil, nil
		},
	}
	route := &handler.Handler{
		DB:            h.DB,
		CtrlClient:    ctrlClient,
		Auditlogs:     h.Auditlogs,
		GitHubAppName: "unkey-app",
		GitHubClient:  github.NewNoop(),
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	// Has create_app but NOT connect_repository.
	rootKey := h.CreateRootKey(workspace.ID, "project.*.create_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        slug(),
	})

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{
		Project: project.ID,
		Name:    "Payments API",
		Slug:    slug(),
		Git:     &openapi.AppGitCreateInput{Repository: ptr.P("unkeyed/unkey")},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}
