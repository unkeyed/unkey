package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
	github "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

func appSlug() string {
	return strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
}

func TestUpdateAppConnectRepository(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
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
	rootKey := h.CreateRootKey(workspace.ID, "app.*.update_app", "app.*.connect_repository")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        appSlug(),
	})
	h.SeedGitHubInstallation(t, workspace.ID, 12345)

	newApp := func(t *testing.T) string {
		t.Helper()
		app := h.CreateApp(seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			Name:        "App",
			Slug:        appSlug(),
		})
		return app.ID
	}

	t.Run("connect defaults to repo default branch", func(t *testing.T) {
		id := newApp(t)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Git:     nullable.NewNullableWithValue(openapi.AppGitUpdateInput{Repository: ptr.P("unkeyed/unkey")}),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body.Data.Git)
		git := res.Body.Data.Git
		require.Equal(t, "unkeyed/unkey", git.Repository)
		require.NotNil(t, git.DefaultBranch)
		require.Equal(t, "main", *git.DefaultBranch)

		conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "unkeyed/unkey", conn.RepositoryFullName)
		require.Equal(t, int64(42), conn.RepositoryID)
		require.Equal(t, int64(12345), conn.InstallationID)

		app, err := db.Query.FindAppById(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "main", app.DefaultBranch)

		requireAuditEvent(ctx, t, h, id, "app.connect_repository")
	})

	t.Run("connect honors explicit default branch", func(t *testing.T) {
		id := newApp(t)
		branch := "develop"

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Git:     nullable.NewNullableWithValue(openapi.AppGitUpdateInput{Repository: ptr.P("unkeyed/unkey"), DefaultBranch: &branch}),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		git := res.Body.Data.Git
		require.Equal(t, "develop", *git.DefaultBranch)

		app, err := db.Query.FindAppById(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "develop", app.DefaultBranch)
	})

	t.Run("replace keeps the existing branch when none passed", func(t *testing.T) {
		// Seed a branch that differs from the repo's GitHub default ("main") so we
		// can prove a replace keeps it instead of silently adopting the default.
		app := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          "App",
			Slug:          appSlug(),
			DefaultBranch: "keep-me",
		})
		id := app.ID

		require.NoError(t, db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
			WorkspaceID:        workspace.ID,
			ProjectID:          project.ID,
			AppID:              id,
			InstallationID:     999,
			RepositoryID:       1,
			RepositoryFullName: "old/repo",
			CreatedAt:          time.Now().UnixMilli(),
			UpdatedAt:          sql.NullInt64{Valid: false},
		}))

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Git:     nullable.NewNullableWithValue(openapi.AppGitUpdateInput{Repository: ptr.P("unkeyed/unkey")}),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		git := res.Body.Data.Git
		require.Equal(t, "unkeyed/unkey", git.Repository, "connection should be replaced, not duplicated")
		require.Equal(t, "keep-me", *git.DefaultBranch, "replace must keep the existing branch, not adopt the repo default")

		conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "unkeyed/unkey", conn.RepositoryFullName)
		require.Equal(t, int64(12345), conn.InstallationID)

		reloaded, err := db.Query.FindAppById(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "keep-me", reloaded.DefaultBranch)
	})

	t.Run("retarget branch only, repository omitted", func(t *testing.T) {
		id := newApp(t)
		require.NoError(t, db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
			WorkspaceID:        workspace.ID,
			ProjectID:          project.ID,
			AppID:              id,
			InstallationID:     12345,
			RepositoryID:       42,
			RepositoryFullName: "unkeyed/unkey",
			CreatedAt:          time.Now().UnixMilli(),
			UpdatedAt:          sql.NullInt64{Valid: false},
		}))

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Git:     nullable.NewNullableWithValue(openapi.AppGitUpdateInput{DefaultBranch: ptr.P("release")}),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		git := res.Body.Data.Git
		require.Equal(t, "unkeyed/unkey", git.Repository, "repository stays connected")
		require.Equal(t, "release", *git.DefaultBranch)

		app, err := db.Query.FindAppById(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "release", app.DefaultBranch)
	})

	t.Run("retarget branch without a connection fails", func(t *testing.T) {
		id := newApp(t)

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
			Project: project.ID,
			App:     id,
			Git:     nullable.NewNullableWithValue(openapi.AppGitUpdateInput{DefaultBranch: ptr.P("release")}),
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})
}

func TestUpdateAppDisconnectRepository(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		GitHubAppName: "unkey-app",
		GitHubClient:  github.NewNoop(),
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.update_app", "app.*.connect_repository")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        appSlug(),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "App",
		Slug:        appSlug(),
	})

	require.NoError(t, db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspace.ID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		InstallationID:     555,
		RepositoryID:       7,
		RepositoryFullName: "unkeyed/unkey",
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{Valid: false},
	}))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
		Git:     nullable.NewNullNullable[openapi.AppGitUpdateInput](),
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Nil(t, res.Body.Data.Git, "response git should be omitted after disconnect")

	_, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), app.ID)
	require.True(t, db.IsNotFound(err), "connection row should be deleted")

	cleared, err := db.Query.FindAppById(ctx, h.DB.RO(), app.ID)
	require.NoError(t, err)
	require.Empty(t, cleared.DefaultBranch, "default branch should be cleared on disconnect")

	requireAuditEvent(ctx, t, h, app.ID, "app.disconnect_repository")
}

func TestUpdateAppConnectRepositoryNotConfigured(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		GitHubAppName: "", // GitHub not configured for this deployment
		GitHubClient:  github.NewNoop(),
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.update_app", "app.*.connect_repository")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        appSlug(),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "App",
		Slug:        appSlug(),
	})

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
		Git:     nullable.NewNullableWithValue(openapi.AppGitUpdateInput{Repository: ptr.P("unkeyed/unkey")}),
	})
	require.GreaterOrEqual(t, res.Status, 500, "unconfigured GitHub connection should fail, received: %s", res.RawBody)
}

func TestUpdateAppConnectRepositoryForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		GitHubAppName: "unkey-app",
		GitHubClient:  github.NewNoop(),
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	// Has update_app but NOT connect_repository.
	rootKey := h.CreateRootKey(workspace.ID, "app.*.update_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments",
		Slug:        appSlug(),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "App",
		Slug:        appSlug(),
	})

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
		Git:     nullable.NewNullableWithValue(openapi.AppGitUpdateInput{Repository: ptr.P("unkeyed/unkey")}),
	})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}

func requireAuditEvent(ctx context.Context, t *testing.T, h *testutil.Harness, targetID, event string) {
	t.Helper()
	logs := h.FindAuditLogsByTargetID(ctx, t, targetID)
	for _, ev := range logs {
		if ev.Event == event {
			return
		}
	}
	require.Failf(t, "audit event not found", "expected event %q for target %s", event, targetID)
}
