package handler_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_connections_update_connection"
)

func TestUpdateConnectionTransitionsSelectionAndPreservesRenamedName(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	require.Equal(t, http.MethodPost, route.Method())
	require.Equal(t, "/v2/connections.updateConnection", route.Path())
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	deployment := seedReadyDeployment(t, h, target)
	id := seedConnection(t, h, environment, target.AppID, "backend", db.InsertConnectionAppTargetParams{
		SelectionMode: db.ConnectionAppTargetsSelectionModeAutomatic,
	})
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID,
		environmentScope(environment)+"#read",
		environmentScope(environment)+"#write",
		appScope(target)+"#read",
		environmentScope(target)+"#read",
		environmentScope(target)+"/deployments/"+deployment.ID+"#read",
	))

	for _, step := range []struct {
		mode                openapi.ConnectionTargetType
		name                *string
		targetEnvironmentID sql.NullString
		targetDeploymentID  sql.NullString
	}{
		{
			mode:                openapi.ConnectionTargetTypeEnvironment,
			name:                new("renamed-backend"),
			targetEnvironmentID: sql.NullString{String: target.ID, Valid: true},
		},
		{
			mode:               openapi.ConnectionTargetTypeDeployment,
			targetDeploymentID: sql.NullString{String: deployment.ID, Valid: true},
		},
		{mode: openapi.ConnectionTargetTypeAutomatic},
	} {
		request := handler.Request{
			ConnectionId:        id,
			Name:                step.name,
			TargetType:          step.mode,
			TargetEnvironmentId: new(uid.New(uid.EnvironmentPrefix)),
			TargetDeploymentId:  new(uid.New(uid.DeploymentPrefix)),
		}
		if step.targetEnvironmentID.Valid {
			request.TargetEnvironmentId = new(step.targetEnvironmentID.String)
		}
		if step.targetDeploymentID.Valid {
			request.TargetDeploymentId = new(step.targetDeploymentID.String)
		}
		response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
		require.Equal(t, http.StatusOK, response.Status, "%s: %s", step.mode, response.RawBody)
		require.Equal(t, id, response.Body.Data.Id)
		require.Equal(t, "renamed-backend", response.Body.Data.Name)
		require.NotEmpty(t, response.Body.Meta.RequestId)
		rows := listConnections(t, h, environment)
		require.Len(t, rows, 1)
		require.Equal(t, id, rows[0].ID)
		require.Equal(t, environment.AppID, rows[0].AppID)
		require.Equal(t, environment.ID, rows[0].EnvironmentID)
		require.Equal(t, target.AppID, rows[0].TargetAppID)
		require.Equal(t, "renamed-backend", rows[0].Name)
		require.Equal(t, db.NullConnectionAppTargetsSelectionMode{
			ConnectionAppTargetsSelectionMode: db.ConnectionAppTargetsSelectionMode(step.mode),
			Valid:                             true,
		}, rows[0].SelectionMode)
		require.Equal(t, step.targetEnvironmentID, rows[0].TargetEnvironmentID)
		require.Equal(t, step.targetDeploymentID, rows[0].TargetDeploymentID)
	}
}

func TestUpdateConnectionMasksUnreadableMissingAndForeignConnections(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	id := seedConnection(t, h, environment, target.AppID, "backend", db.InsertConnectionAppTargetParams{
		SelectionMode: db.ConnectionAppTargetsSelectionModeAutomatic,
	})
	before := listConnections(t, h, environment)
	request := handler.Request{
		ConnectionId: uid.New(uid.ConnectionPrefix),
		Name:         new("renamed-backend"),
		TargetType:   openapi.ConnectionTargetTypeAutomatic,
	}
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/connection_not_found", missing.Body.Error.Type)
	request.ConnectionId = id

	for _, test := range []struct {
		name        string
		workspaceID string
		permissions []string
		status      int
	}{
		{
			name:        "no caller access",
			workspaceID: environment.WorkspaceID,
			permissions: []string{appScope(target) + "#read"},
			status:      http.StatusNotFound,
		},
		{
			name:        "write without read",
			workspaceID: environment.WorkspaceID,
			permissions: []string{environmentScope(environment) + "#write", appScope(target) + "#read"},
			status:      http.StatusNotFound,
		},
		{
			name:        "read without write",
			workspaceID: environment.WorkspaceID,
			permissions: []string{environmentScope(environment) + "#read", appScope(target) + "#read"},
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
			response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, test.status, response.Status, "%s", response.RawBody)
			if test.status == http.StatusNotFound {
				require.Equal(t, missing.Body.Error, response.Body.Error)
			}
			require.Equal(t, before, listConnections(t, h, environment))
		})
	}
}

func TestUpdateConnectionRejectsRenamesWithoutChangingSelection(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	targetOther := seedTarget(t, h, environment, "other-target")
	id := seedConnection(t, h, environment, target.AppID, "backend", db.InsertConnectionAppTargetParams{
		SelectionMode:       db.ConnectionAppTargetsSelectionModeEnvironment,
		TargetEnvironmentID: sql.NullString{String: target.ID, Valid: true},
	})
	seedConnection(t, h, environment, targetOther.AppID, "taken", db.InsertConnectionAppTargetParams{
		SelectionMode: db.ConnectionAppTargetsSelectionModeAutomatic,
	})
	before := listConnections(t, h, environment)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))

	for _, test := range []struct {
		name   string
		status int
	}{
		{name: "taken", status: http.StatusConflict},
		{name: "self", status: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{
				ConnectionId: id,
				Name:         new(test.name),
				TargetType:   openapi.ConnectionTargetTypeAutomatic,
			})
			require.Equal(t, test.status, response.Status, "%s", response.RawBody)
			require.Equal(t, before, listConnections(t, h, environment))
		})
	}
}

func TestUpdateConnectionRequiresTargetAppAndSelectedTargetRead(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	deployment := seedReadyDeployment(t, h, target)
	id := seedConnection(t, h, environment, target.AppID, "backend", db.InsertConnectionAppTargetParams{
		SelectionMode: db.ConnectionAppTargetsSelectionModeAutomatic,
	})
	before := listConnections(t, h, environment)
	for _, test := range []struct {
		name        string
		mode        openapi.ConnectionTargetType
		permissions []string
		errorType   string
	}{
		{
			name:      "target app",
			mode:      openapi.ConnectionTargetTypeAutomatic,
			errorType: "https://unkey.com/docs/errors/unkey/data/app_not_found",
		},
		{
			name:        "target environment",
			mode:        openapi.ConnectionTargetTypeEnvironment,
			permissions: []string{appScope(target) + "#read"},
			errorType:   "https://unkey.com/docs/errors/unkey/data/environment_not_found",
		},
		{
			name:        "target deployment",
			mode:        openapi.ConnectionTargetTypeDeployment,
			permissions: []string{appScope(target) + "#read"},
			errorType:   "https://unkey.com/docs/errors/unkey/data/deployment_not_found",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			permissions := append([]string{environmentScope(environment) + "#read", environmentScope(environment) + "#write"}, test.permissions...)
			headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, permissions...))
			response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
				ConnectionId:        id,
				Name:                new("renamed-backend"),
				TargetType:          test.mode,
				TargetEnvironmentId: new(target.ID),
				TargetDeploymentId:  new(deployment.ID),
			})
			require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
			require.Equal(t, test.errorType, response.Body.Error.Type)
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

func seedTarget(t *testing.T, h *testutil.Harness, environment db.Environment, slug string) db.Environment {
	t.Helper()
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		Name:        "Target " + slug,
		Slug:        slug,
	})
	return h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: app.WorkspaceID,
		ProjectID:   app.ProjectID,
		AppID:       app.ID,
		Slug:        "production",
	})
}

func seedReadyDeployment(t *testing.T, h *testutil.Harness, environment db.Environment) db.Deployment {
	t.Helper()
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		DesiredState:  mysqltype.DeploymentsDesiredStateRunning,
	})
	require.NoError(t, db.Query.UpdateDeploymentFirstReadyAt(t.Context(), h.DB.RW(), db.UpdateDeploymentFirstReadyAtParams{
		ID:           deployment.ID,
		FirstReadyAt: sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: true},
	}))
	return deployment
}

func seedConnection(t *testing.T, h *testutil.Harness, environment db.Environment, targetID, name string, selection db.InsertConnectionAppTargetParams) string {
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
		Name:          name,
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
	return appScope(environment) + "/environments/" + environment.ID
}

func appScope(environment db.Environment) string {
	return "unkey:v1:" + environment.WorkspaceID + ":projects/" + environment.ProjectID + "/apps/" + environment.AppID
}

func authHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}
