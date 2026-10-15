package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
)

func TestDeleteAppSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	restateClient, deletes := newRecordingRestate(t)

	route := &handler.Handler{
		DB:      h.DB,
		Restate: restateClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.delete_app")
	rootKeyID, err := db.Query.FindKeyIDByHash(ctx, h.DB.RO(), hash.Sha256(rootKey))
	require.NoError(t, err)
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
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Doomed",
		Slug:        appSlug,
	})
	topologies := seedDeleteTopologies(t, h, workspace.ID, project.ID, app.ID)
	sibling := h.CreateApp(seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: workspace.ID, ProjectID: project.ID,
		Name: "Sibling", Slug: "sibling",
	})
	siblingTopologies := seedDeleteTopologies(t, h, workspace.ID, project.ID, sibling.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
	})
	require.Equal(t, 202, res.Status, "expected 202, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	observed := testutil.Receive(t, deletes, 10*time.Second)
	require.Equal(t, app.ID, observed.virtualObjectKey)
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, observed.request.GetActor().GetType())
	require.Equal(t, rootKeyID, observed.request.GetActor().GetId())
	require.NotEmpty(t, observed.request.GetCorrelationId())

	var deletedAt sql.NullInt64
	require.NoError(t, h.DB.RW().QueryRowContext(ctx, "SELECT deleted_at FROM apps WHERE id = ?", app.ID).Scan(&deletedAt))
	require.True(t, deletedAt.Valid)
	for deploymentID, before := range topologies {
		var updatedAt, running int64
		require.NoError(t, h.DB.RW().QueryRowContext(ctx, "SELECT MIN(COALESCE(updated_at, 0)), SUM(desired_status = 'running') FROM deployment_topology WHERE deployment_id = ?", deploymentID).Scan(&updatedAt, &running))
		require.Greater(t, updatedAt, before)
		require.Zero(t, running, "deleted apps must release compute quota before the worker finishes")
	}
	for deploymentID, before := range siblingTopologies {
		var updatedAt, stopped int64
		require.NoError(t, h.DB.RW().QueryRowContext(ctx, "SELECT COALESCE(MAX(updated_at), 0), SUM(desired_status = 'stopped') FROM deployment_topology WHERE deployment_id = ?", deploymentID).Scan(&updatedAt, &stopped))
		require.Equal(t, before, updatedAt)
		require.Zero(t, stopped)
	}
	again := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.ID, App: app.ID})
	require.Equal(t, http.StatusNotFound, again.Status)
}

func seedDeleteTopologies(t *testing.T, h *testutil.Harness, workspaceID, projectID, appID string) map[string]int64 {
	t.Helper()
	before := make(map[string]int64)
	for range 2 {
		environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspaceID, ProjectID: projectID, AppID: appID,
			Slug: strings.ToLower(strings.ReplaceAll(uid.New("env"), "_", "-")),
		})
		for index := range 2 {
			deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
				ID: uid.New(uid.DeploymentPrefix), WorkspaceID: workspaceID, ProjectID: projectID, AppID: appID, EnvironmentID: environment.ID,
			})
			initial := sql.NullInt64{Valid: index == 1, Int64: time.Now().Add(time.Hour).UnixMilli()}
			if initial.Valid {
				before[deployment.ID] = initial.Int64
			} else {
				before[deployment.ID] = 0
			}
			for range 2 {
				_, err := h.DB.RW().ExecContext(t.Context(), `INSERT INTO deployment_topology
					(workspace_id, deployment_id, region_id, desired_status, created_at, updated_at)
					VALUES (?, ?, ?, 'running', ?, ?)`, workspaceID, deployment.ID, uid.New("region"), time.Now().UnixMilli(), initial)
				require.NoError(t, err)
			}
		}
	}
	return before
}
