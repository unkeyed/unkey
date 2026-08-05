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
	"github.com/unkeyed/unkey/pkg/uid"
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
		Git:     &openapi.AppGitCreateInput{Repository: "unkeyed/unkey"},
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, appID, res.Body.Data.AppId)
	require.Len(t, ctrlClient.CreateAppCalls, 1)
	require.NotNil(t, ctrlClient.CreateAppCalls[0].GetGithub())

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
		Git:     &openapi.AppGitCreateInput{Repository: "unkeyed/unkey"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}
