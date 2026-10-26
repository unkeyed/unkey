package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_connections_list_retained_connections"
)

func TestListRetainedConnectionsCountsStoppedAndFailedSnapshotsWithoutTargetSettingsOrCapabilities(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	environment := seedEnvironment(t, h, h.Resources().UserWorkspace.ID)
	target := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		Name:        "Current target name",
		Slug:        "target",
	})
	id := uid.New(uid.ConnectionPrefix)
	require.NoError(t, db.Query.InsertAppConnection(t.Context(), h.DB.RW(), db.InsertAppConnectionParams{
		ID:            id,
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		ResourceType:  db.AppConnectionsResourceTypeApp,
		ResourceID:    target.ID,
		Name:          "current-default-name",
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	for _, test := range []struct {
		name         string
		status       mysqltype.DeploymentsStatus
		desiredState mysqltype.DeploymentsDesiredState
	}{
		{
			name:         "saved-before-target-rename",
			status:       mysqltype.DeploymentsStatusReady,
			desiredState: mysqltype.DeploymentsDesiredStateStopped,
		},
		{
			name:         "different-saved-name",
			status:       mysqltype.DeploymentsStatusFailed,
			desiredState: mysqltype.DeploymentsDesiredStateRunning,
		},
	} {
		deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   environment.WorkspaceID,
			ProjectID:     environment.ProjectID,
			AppID:         environment.AppID,
			EnvironmentID: environment.ID,
			Status:        test.status,
			DesiredState:  test.desiredState,
		})
		require.Empty(t, deployment.Capabilities)
		seedSnapshot(t, h, deployment, id, target.ID, test.name)
	}

	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, environmentScope(environment)+"#read"))
	request := handler.Request{Project: environment.ProjectID, App: environment.AppID, Environment: environment.ID}
	current := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, current.Status, "%s", current.RawBody)
	require.NotNil(t, current.Body.Data)
	require.Empty(t, current.Body.Data)

	require.NoError(t, db.Query.DeleteAppConnection(t.Context(), h.DB.RW(), db.DeleteAppConnectionParams{
		ID:          id,
		WorkspaceID: environment.WorkspaceID,
	}))
	for _, project := range []string{environment.ProjectID, "project"} {
		for _, app := range []string{environment.AppID, "caller"} {
			for _, environmentID := range []string{environment.ID, environment.Slug} {
				t.Run(project+"/"+app+"/"+environmentID, func(t *testing.T) {
					response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
						Project:     project,
						App:         app,
						Environment: environmentID,
					})
					require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
					require.NotEmpty(t, response.Body.Meta.RequestId)
					require.Equal(t, []openapi.RetainedAppConnection{{
						Id:            id,
						TargetAppName: target.Name,
						Deployments:   2,
					}}, response.Body.Data)
				})
			}
		}
	}
}

func TestListRetainedConnectionsExcludesOtherCallerEnvironmentsAndWorkspaces(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	environment := seedEnvironment(t, h, h.Resources().UserWorkspace.ID)
	environmentOther := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		AppID:       environment.AppID,
		Slug:        "other-preview",
	})
	environmentForeign := seedEnvironment(t, h, h.CreateWorkspace().ID)
	target := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		Name:        "Target",
		Slug:        "target",
	})
	id := uid.New(uid.ConnectionPrefix)
	for _, caller := range []db.Environment{environmentOther, environmentForeign} {
		deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   caller.WorkspaceID,
			ProjectID:     caller.ProjectID,
			AppID:         caller.AppID,
			EnvironmentID: caller.ID,
		})
		seedSnapshot(t, h, deployment, id, target.ID, "same-connection")
		seedSnapshot(t, h, deployment, uid.New(uid.ConnectionPrefix), target.ID, "other-connection")
	}
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, environmentScope(environment)+"#read"))
	request := handler.Request{Project: environment.ProjectID, App: environment.AppID, Environment: environment.ID}
	empty := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, empty.Status, "%s", empty.RawBody)
	require.NotNil(t, empty.Body.Data)
	require.Empty(t, empty.Body.Data)

	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	seedSnapshot(t, h, deployment, id, target.ID, "caller-connection")
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Equal(t, []openapi.RetainedAppConnection{{
		Id:            id,
		TargetAppName: target.Name,
		Deployments:   1,
	}}, response.Body.Data)
}

func TestListRetainedConnectionsRequiresReadAndMasksMissingUnreadableAndForeignEnvironments(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	environment := seedEnvironment(t, h, h.Resources().UserWorkspace.ID)
	request := handler.Request{Project: environment.ProjectID, App: environment.AppID, Environment: environment.ID}
	requestMissing := request
	requestMissing.Environment = uid.New(uid.EnvironmentPrefix)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, requestMissing)
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/environment_not_found", missing.Body.Error.Type)

	for _, test := range []struct {
		name        string
		workspaceID string
		permissions []string
	}{
		{
			name:        "no caller access",
			workspaceID: environment.WorkspaceID,
		},
		{
			name:        "write without read",
			workspaceID: environment.WorkspaceID,
			permissions: []string{environmentScope(environment) + "#write"},
		},
		{
			name:        "foreign workspace",
			workspaceID: h.Resources().RootWorkspace.ID,
			permissions: []string{"unkey:v1:" + h.Resources().RootWorkspace.ID + ":**#*"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(test.workspaceID, test.permissions...))
			response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
			require.Equal(t, missing.Body.Error, response.Body.Error)
		})
	}
}

func seedEnvironment(t *testing.T, h *testutil.Harness, workspaceID string) db.Environment {
	t.Helper()
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Project",
		Slug:        "project",
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "Caller",
		Slug:        "caller",
	})
	return h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
	})
}

func seedSnapshot(t *testing.T, h *testutil.Harness, deployment db.Deployment, connectionID, targetID, name string) {
	t.Helper()
	require.NoError(t, db.Query.InsertDeploymentConnection(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionParams{
		DeploymentID:  deployment.ID,
		ConnectionID:  connectionID,
		WorkspaceID:   deployment.WorkspaceID,
		ProjectID:     deployment.ProjectID,
		AppID:         deployment.AppID,
		EnvironmentID: deployment.EnvironmentID,
		ResourceType:  db.DeploymentConnectionsResourceTypeApp,
		ResourceID:    targetID,
		Name:          name,
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
}

func environmentScope(environment db.Environment) string {
	return "unkey:v1:" + environment.WorkspaceID + ":projects/" + environment.ProjectID + "/apps/" + environment.AppID + "/environments/" + environment.ID
}

func authHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}
