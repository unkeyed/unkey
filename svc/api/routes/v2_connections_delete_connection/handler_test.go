package handler_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_connections_delete_connection"
)

func TestDeleteConnectionPreservesOtherEnvironmentAndSavedSnapshotWithoutTargetRead(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	require.Equal(t, http.MethodPost, route.Method())
	require.Equal(t, "/v2/connections.deleteConnection", route.Path())
	environment := seedEnvironment(t, h)
	target := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		Name:        "Target",
		Slug:        "target",
	})
	targetEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: target.WorkspaceID,
		ProjectID:   target.ProjectID,
		AppID:       target.ID,
		Slug:        "production",
	})
	targetDeployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   target.WorkspaceID,
		ProjectID:     target.ProjectID,
		AppID:         target.ID,
		EnvironmentID: targetEnvironment.ID,
	})
	id := seedConnection(t, h, environment, target.ID, db.InsertConnectionAppTargetParams{
		SelectionMode:       db.ConnectionAppTargetsSelectionModeEnvironment,
		TargetEnvironmentID: sql.NullString{String: targetEnvironment.ID, Valid: true},
	})
	environmentOther := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		AppID:       environment.AppID,
		Slug:        "another-preview",
	})
	seedConnection(t, h, environmentOther, target.ID, db.InsertConnectionAppTargetParams{
		SelectionMode:      db.ConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: targetDeployment.ID, Valid: true},
	})
	otherBefore := listConnections(t, h, environmentOther)
	require.Len(t, otherBefore, 1)
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	require.NoError(t, db.Query.InsertDeploymentConnection(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionParams{
		DeploymentID:  deployment.ID,
		ConnectionID:  id,
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		ResourceType:  db.DeploymentConnectionsResourceTypeApp,
		ResourceID:    target.ID,
		Name:          "saved-backend",
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertDeploymentConnectionAppTarget(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionAppTargetParams{
		DeploymentID:       deployment.ID,
		ConnectionID:       id,
		SelectionMode:      db.DeploymentConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: targetDeployment.ID, Valid: true},
	}))
	savedParams := db.ListSavedAppConnectionsParams{
		WorkspaceID:  environment.WorkspaceID,
		DeploymentID: deployment.ID,
	}
	savedBefore, err := db.Query.ListSavedAppConnections(t.Context(), h.DB.RW(), savedParams)
	require.NoError(t, err)
	require.Len(t, savedBefore, 1)
	require.Equal(t, id, savedBefore[0].ConnectionID)
	require.Equal(t, "saved-backend", savedBefore[0].Name)
	require.Equal(t, target.ID, savedBefore[0].ResourceID)
	require.Equal(t, db.DeploymentConnectionAppTargetsSelectionModeDeployment, savedBefore[0].SelectionMode)
	require.Equal(t, sql.NullString{String: targetDeployment.ID, Valid: true}, savedBefore[0].TargetDeploymentID)
	require.False(t, savedBefore[0].TargetEnvironmentID.Valid)

	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID,
		environmentScope(environment)+"#read",
		environmentScope(environment)+"#write",
	))
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ConnectionId: id})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Equal(t, id, response.Body.Data.Id)
	require.NotEmpty(t, response.Body.Meta.RequestId)
	require.Empty(t, listConnections(t, h, environment))
	require.Equal(t, otherBefore, listConnections(t, h, environmentOther))
	savedAfter, err := db.Query.ListSavedAppConnections(t.Context(), h.DB.RW(), savedParams)
	require.NoError(t, err)
	require.Equal(t, savedBefore, savedAfter)
}

func TestDeleteConnectionMasksUnreadableMissingAndForeignConnectionsAndRejectsReadOnly(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		Name:        "Target",
		Slug:        "target",
	})
	id := seedConnection(t, h, environment, target.ID, db.InsertConnectionAppTargetParams{
		SelectionMode: db.ConnectionAppTargetsSelectionModeAutomatic,
	})
	before := listConnections(t, h, environment)
	require.Len(t, before, 1)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		ConnectionId: uid.New(uid.ConnectionPrefix),
	})
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/connection_not_found", missing.Body.Error.Type)

	for _, test := range []struct {
		name        string
		workspaceID string
		permissions []string
		status      int
	}{
		{
			name:        "no caller access",
			workspaceID: environment.WorkspaceID,
			status:      http.StatusNotFound,
		},
		{
			name:        "write without read",
			workspaceID: environment.WorkspaceID,
			permissions: []string{environmentScope(environment) + "#write"},
			status:      http.StatusNotFound,
		},
		{
			name:        "read without write",
			workspaceID: environment.WorkspaceID,
			permissions: []string{environmentScope(environment) + "#read"},
			status:      http.StatusForbidden,
		},
		{
			name:        "foreign workspace",
			workspaceID: h.Resources().RootWorkspace.ID,
			permissions: []string{"unkey:v1:" + h.Resources().RootWorkspace.ID + ":**#*"},
			status:      http.StatusNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(test.workspaceID, test.permissions...))
			response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{ConnectionId: id})
			require.Equal(t, test.status, response.Status, "%s", response.RawBody)
			if test.status == http.StatusNotFound {
				require.Equal(t, missing.Body.Error, response.Body.Error)
			}
			require.Equal(t, before, listConnections(t, h, environment))
		})
	}
}

func seedEnvironment(t *testing.T, h *testutil.Harness) db.Environment {
	t.Helper()
	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Project",
		Slug:        "project",
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Caller",
		Slug:        "caller",
	})
	return h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
	})
}

func seedConnection(t *testing.T, h *testutil.Harness, environment db.Environment, targetID string, selection db.InsertConnectionAppTargetParams) string {
	t.Helper()
	id := uid.New(uid.ConnectionPrefix)
	require.NoError(t, db.Query.InsertAppConnection(t.Context(), h.DB.RW(), db.InsertAppConnectionParams{
		ID:            id,
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		ResourceType:  db.AppConnectionsResourceTypeApp,
		ResourceID:    targetID,
		Name:          "backend",
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	selection.ConnectionID = id
	require.NoError(t, db.Query.InsertConnectionAppTarget(t.Context(), h.DB.RW(), selection))
	return id
}

func listConnections(t *testing.T, h *testutil.Harness, environment db.Environment) []db.ListAppConnectionsRow {
	t.Helper()
	rows, err := db.Query.ListAppConnections(t.Context(), h.DB.RW(), db.ListAppConnectionsParams{
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		Limit:         100,
	})
	require.NoError(t, err)
	return rows
}

func environmentScope(environment db.Environment) string {
	return "unkey:v1:" + environment.WorkspaceID + ":projects/" + environment.ProjectID + "/apps/" + environment.AppID + "/environments/" + environment.ID
}

func authHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}
