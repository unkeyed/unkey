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
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	github "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
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
		require.Equal(t, "main", conn.DefaultBranch.String)

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

		conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "develop", conn.DefaultBranch.String)
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
			DefaultBranch: "stale-app-value",
		})
		id := app.ID

		require.NoError(t, db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
			WorkspaceID:        workspace.ID,
			ProjectID:          project.ID,
			AppID:              id,
			InstallationID:     999,
			RepositoryID:       1,
			RepositoryFullName: "old/repo",
			DefaultBranch:      sql.NullString{Valid: true, String: "keep-me"},
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
		require.Equal(t, "keep-me", conn.DefaultBranch.String)
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
			DefaultBranch:      sql.NullString{Valid: true, String: "main"},
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

		conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Equal(t, "release", conn.DefaultBranch.String)

		app, err := db.Query.FindAppById(ctx, h.DB.RO(), id)
		require.NoError(t, err)
		require.Empty(t, app.DefaultBranch, "branch updates must not write the legacy app column")
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
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "App",
		Slug:          appSlug(),
		DefaultBranch: "stale-app-value",
	})

	require.NoError(t, db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspace.ID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		InstallationID:     555,
		RepositoryID:       7,
		RepositoryFullName: "unkeyed/unkey",
		DefaultBranch:      sql.NullString{Valid: true, String: "main"},
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

	unchanged, err := db.Query.FindAppById(ctx, h.DB.RO(), app.ID)
	require.NoError(t, err)
	require.Equal(t, "stale-app-value", unchanged.DefaultBranch, "disconnect must not write the legacy app column")

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

func TestUpdateAppOCIImageWithAppSettings(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockAppClient{
		UpdateOciImageSourceFunc: func(_ context.Context, _ *ctrlv1.UpdateOciImageSourceRequest) (*ctrlv1.UpdateOciImageSourceResponse, error) {
			return &ctrlv1.UpdateOciImageSourceResponse{ImageReference: "index.docker.io/library/nginx:1.27"}, nil
		},
	}
	route := &handler.Handler{
		DB:         h.DB,
		Auditlogs:  h.Auditlogs,
		CtrlClient: ctrlClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.update_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "OCI",
		Slug:        appSlug(),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "OCI app",
		Slug:        appSlug(),
		SourceType:  db.AppsSourceTypeOci,
	})
	updatedName := "Updated OCI app"
	updatedSlug := appSlug()

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project:          project.ID,
		App:              app.ID,
		Name:             &updatedName,
		Slug:             &updatedSlug,
		DeleteProtection: ptr.P(true),
		Oci: &openapi.AppOCI{
			Image: "nginx:1.27",
		},
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, "oci", string(res.Body.Data.SourceType))
	require.NotNil(t, res.Body.Data.Oci)
	require.Equal(t, "index.docker.io/library/nginx:1.27", res.Body.Data.Oci.Image)
	require.Nil(t, res.Body.Data.Git)
	require.Equal(t, updatedName, res.Body.Data.Name)
	require.Equal(t, updatedSlug, res.Body.Data.Slug)
	require.True(t, res.Body.Data.DeleteProtection)
	require.Len(t, ctrlClient.UpdateOciImageSourceCalls, 1)
	call := ctrlClient.UpdateOciImageSourceCalls[0]
	require.Equal(t, workspace.ID, call.GetWorkspaceId())
	require.Equal(t, app.ID, call.GetAppId())
	require.Equal(t, "nginx:1.27", call.GetImageReference())
	require.NotNil(t, call.GetActor())

	updatedApp, err := db.Query.FindAppById(context.Background(), h.DB.RO(), app.ID)
	require.NoError(t, err)
	require.Equal(t, updatedName, updatedApp.Name)
	require.Equal(t, updatedSlug, updatedApp.Slug)
	require.True(t, updatedApp.DeleteProtection.Bool)
}

func TestUpdateAppRejectsSourceSwitching(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockAppClient{}
	route := &handler.Handler{
		DB:         h.DB,
		Auditlogs:  h.Auditlogs,
		CtrlClient: ctrlClient,
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
		Name:        "Sources",
		Slug:        appSlug(),
	})
	createApp := func(t *testing.T, sourceType db.AppsSourceType) db.App {
		t.Helper()
		return h.CreateApp(seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			Name:        "App",
			Slug:        appSlug(),
			SourceType:  sourceType,
		})
	}

	t.Run("git update on OCI app", func(t *testing.T) {
		app := createApp(t, db.AppsSourceTypeOci)
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
			Project: project.ID,
			App:     app.ID,
			Git:     nullable.NewNullNullable[openapi.AppGitUpdateInput](),
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})

	for _, sourceType := range []db.AppsSourceType{db.AppsSourceTypeGit, db.AppsSourceTypeUnknown} {
		t.Run("image update on "+string(sourceType)+" app", func(t *testing.T) {
			app := createApp(t, sourceType)
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
				Project: project.ID,
				App:     app.ID,
				Oci:     &openapi.AppOCI{Image: "ghcr.io/acme/api:v2"},
			})
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	t.Run("git and image together", func(t *testing.T) {
		app := createApp(t, db.AppsSourceTypeOci)
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
			Project: project.ID,
			App:     app.ID,
			Git:     nullable.NewNullNullable[openapi.AppGitUpdateInput](),
			Oci:     &openapi.AppOCI{Image: "ghcr.io/acme/api:v2"},
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})

	require.Empty(t, ctrlClient.UpdateOciImageSourceCalls)
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
