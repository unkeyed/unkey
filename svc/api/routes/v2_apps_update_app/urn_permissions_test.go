package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
)

// TestUpdateAppAuthorizesCanonicalWriteForSettings guarantees the canonical
// app write grant can update app settings without legacy permissions.
func TestUpdateAppAuthorizesCanonicalWriteForSettings(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Canonical Update",
		Slug:        appSlug(),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Before",
		Slug:        appSlug(),
	})
	permission := fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s#write", workspace.ID, project.ID, app.ID)
	rootKey := h.CreateRootKey(workspace.ID, permission)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	name := "After"

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
		Name:    &name,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	updated, err := db.Query.FindAppById(context.Background(), h.DB.RO(), app.ID)
	require.NoError(t, err)
	require.Equal(t, name, updated.Name)
}

// TestUpdateAppAuthorizesCanonicalWriteForGitDisconnect guarantees the
// canonical app write grant can disconnect Git without legacy permissions.
func TestUpdateAppAuthorizesCanonicalWriteForGitDisconnect(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Canonical Git Disconnect",
		Slug:        appSlug(),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Git App",
		Slug:        appSlug(),
	})
	require.NoError(t, db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspace.ID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		InstallationID:     123,
		RepositoryID:       456,
		RepositoryFullName: "unkeyed/unkey",
		DefaultBranch:      sql.NullString{Valid: true, String: "main"},
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{},
	}))

	permission := fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s#write", workspace.ID, project.ID, app.ID)
	rootKey := h.CreateRootKey(workspace.ID, permission)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
		Git:     nullable.NewNullNullable[openapi.AppGitUpdateInput](),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	_, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), app.ID)
	require.True(t, db.IsNotFound(err), "connection row should be deleted")
}

// TestUpdateAppRejectsMismatchedCanonicalWriteWithoutMutation guarantees a
// canonical grant cannot cross app, project, workspace, or action boundaries.
func TestUpdateAppRejectsMismatchedCanonicalWriteWithoutMutation(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Canonical Boundaries",
		Slug:        appSlug(),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Unchanged",
		Slug:        appSlug(),
	})
	require.NoError(t, db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspace.ID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		InstallationID:     123,
		RepositoryID:       456,
		RepositoryFullName: "unkeyed/unkey",
		DefaultBranch:      sql.NullString{Valid: true, String: "main"},
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{},
	}))

	testCases := []struct {
		name       string
		permission string
	}{
		{
			name:       "wrong app",
			permission: fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s#write", workspace.ID, project.ID, uid.New(uid.AppPrefix)),
		},
		{
			name:       "wrong project",
			permission: fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s#write", workspace.ID, uid.New(uid.ProjectPrefix), app.ID),
		},
		{
			name:       "wrong workspace",
			permission: fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s#write", uid.New(uid.WorkspacePrefix), project.ID, app.ID),
		},
		{
			name:       "wrong action",
			permission: fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s#read", workspace.ID, project.ID, app.ID),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, testCase.permission)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			name := "Unauthorized change"

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project: project.ID,
				App:     app.ID,
				Name:    &name,
				Git:     nullable.NewNullNullable[openapi.AppGitUpdateInput](),
			})
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)

			unchanged, err := db.Query.FindAppById(ctx, h.DB.RO(), app.ID)
			require.NoError(t, err)
			require.Equal(t, "Unchanged", unchanged.Name)

			connection, err := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), app.ID)
			require.NoError(t, err)
			require.Equal(t, "unkeyed/unkey", connection.RepositoryFullName)
		})
	}
}
