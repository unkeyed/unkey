package handler_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_connections"
)

func TestListConnectionsReturnsSavedSelectionsIndependentlyOfCurrentDefaults(t *testing.T) {
	h, route, deployment := setup(t)
	target := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: deployment.WorkspaceID,
		ProjectID:   deployment.ProjectID,
		Name:        "Current target name",
		Slug:        "target",
	})
	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: target.WorkspaceID,
		ProjectID:   target.ProjectID,
		AppID:       target.ID,
		Slug:        "production",
	})
	targetDeploymentID := uid.New(uid.DeploymentPrefix)
	automaticID := seedSnapshot(t, h, deployment, target.ID, "a-saved", db.InsertDeploymentConnectionAppTargetParams{
		SelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic,
	})
	environmentID := seedSnapshot(t, h, deployment, target.ID, "b-saved", db.InsertDeploymentConnectionAppTargetParams{
		SelectionMode:       db.DeploymentConnectionAppTargetsSelectionModeEnvironment,
		TargetEnvironmentID: sql.NullString{String: environment.ID, Valid: true},
	})
	deploymentID := seedSnapshot(t, h, deployment, target.ID, "c-saved", db.InsertDeploymentConnectionAppTargetParams{
		SelectionMode:      db.DeploymentConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: targetDeploymentID, Valid: true},
	})
	require.NoError(t, db.Query.InsertAppConnection(t.Context(), h.DB.RW(), db.InsertAppConnectionParams{
		ID:            automaticID,
		WorkspaceID:   deployment.WorkspaceID,
		ProjectID:     deployment.ProjectID,
		AppID:         deployment.AppID,
		EnvironmentID: deployment.EnvironmentID,
		ResourceType:  db.AppConnectionsResourceTypeApp,
		ResourceID:    uid.New(uid.AppPrefix),
		Name:          "z-current-default",
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertConnectionAppTarget(t.Context(), h.DB.RW(), db.InsertConnectionAppTargetParams{
		ConnectionID:       automaticID,
		SelectionMode:      db.ConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: uid.New(uid.DeploymentPrefix), Valid: true},
	}))
	headers := readHeaders(h, deployment)
	want := []openapi.DeploymentConnectionSummary{
		{
			Id:                    automaticID,
			TargetAppId:           target.ID,
			TargetType:            openapi.ConnectionTargetTypeAutomatic,
			TargetAppName:         nullable.NewNullableWithValue(target.Name),
			TargetEnvironmentId:   nullable.NewNullNullable[string](),
			TargetEnvironmentSlug: nullable.NewNullNullable[string](),
			TargetDeploymentId:    nullable.NewNullNullable[string](),
		},
		{
			Id:                    environmentID,
			TargetAppId:           target.ID,
			TargetType:            openapi.ConnectionTargetTypeEnvironment,
			TargetAppName:         nullable.NewNullableWithValue(target.Name),
			TargetEnvironmentId:   nullable.NewNullableWithValue(environment.ID),
			TargetEnvironmentSlug: nullable.NewNullableWithValue(environment.Slug),
			TargetDeploymentId:    nullable.NewNullNullable[string](),
		},
		{
			Id:                    deploymentID,
			TargetAppId:           target.ID,
			TargetType:            openapi.ConnectionTargetTypeDeployment,
			TargetAppName:         nullable.NewNullableWithValue(target.Name),
			TargetDeploymentId:    nullable.NewNullableWithValue(targetDeploymentID),
			TargetEnvironmentId:   nullable.NewNullNullable[string](),
			TargetEnvironmentSlug: nullable.NewNullNullable[string](),
		},
	}
	for _, deleted := range []bool{false, true} {
		if deleted {
			require.NoError(t, db.Query.DeleteAppConnection(t.Context(), h.DB.RW(), db.DeleteAppConnectionParams{
				ID:          automaticID,
				WorkspaceID: deployment.WorkspaceID,
			}))
		}
		response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{DeploymentId: deployment.ID})
		require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
		require.NotEmpty(t, response.Body.Meta.RequestId)
		require.ElementsMatch(t, want, response.Body.Data)
		require.False(t, response.Body.Pagination.HasMore)
	}
}

func TestListConnectionsPreservesMissingTargetIDsWithNullLabels(t *testing.T) {
	h, route, deployment := setup(t)
	target := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: deployment.WorkspaceID,
		ProjectID:   deployment.ProjectID,
		Name:        "Existing app",
		Slug:        "existing",
	})
	environmentID := uid.New(uid.EnvironmentPrefix)
	environmentMissingID := seedSnapshot(t, h, deployment, target.ID, "a-missing-environment", db.InsertDeploymentConnectionAppTargetParams{
		SelectionMode:       db.DeploymentConnectionAppTargetsSelectionModeEnvironment,
		TargetEnvironmentID: sql.NullString{String: environmentID, Valid: true},
	})
	appMissing := uid.New(uid.AppPrefix)
	appMissingID := seedSnapshot(t, h, deployment, appMissing, "b-missing-app", db.InsertDeploymentConnectionAppTargetParams{
		SelectionMode:       db.DeploymentConnectionAppTargetsSelectionModeEnvironment,
		TargetEnvironmentID: sql.NullString{String: environmentID, Valid: true},
	})
	headers := readHeaders(h, deployment)
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{DeploymentId: deployment.ID})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.ElementsMatch(t, []openapi.DeploymentConnectionSummary{
		{
			Id:                    environmentMissingID,
			TargetAppId:           target.ID,
			TargetType:            openapi.ConnectionTargetTypeEnvironment,
			TargetAppName:         nullable.NewNullableWithValue(target.Name),
			TargetEnvironmentId:   nullable.NewNullableWithValue(environmentID),
			TargetEnvironmentSlug: nullable.NewNullNullable[string](),
			TargetDeploymentId:    nullable.NewNullNullable[string](),
		},
		{
			Id:                    appMissingID,
			TargetAppId:           appMissing,
			TargetType:            openapi.ConnectionTargetTypeEnvironment,
			TargetAppName:         nullable.NewNullNullable[string](),
			TargetEnvironmentId:   nullable.NewNullableWithValue(environmentID),
			TargetEnvironmentSlug: nullable.NewNullNullable[string](),
			TargetDeploymentId:    nullable.NewNullNullable[string](),
		},
	}, response.Body.Data)
}

func TestListConnectionsPaginatesReadableTargetApps(t *testing.T) {
	h, route, deployment := setup(t)
	permissions := []string{deploymentScope(deployment) + "#read"}
	var want []openapi.DeploymentConnectionSummary
	for _, name := range []string{"first", "second", "third", "hidden"} {
		targetID := uid.New(uid.AppPrefix)
		id := seedSnapshot(t, h, deployment, targetID, name, db.InsertDeploymentConnectionAppTargetParams{
			SelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic,
		})
		if name != "hidden" {
			permissions = append(permissions, "unkey:v1:"+deployment.WorkspaceID+":projects/"+deployment.ProjectID+"/apps/"+targetID+"#read")
			want = append(want, automaticSummary(id, targetID))
		}
	}
	headers := authHeaders(h.CreateRootKey(deployment.WorkspaceID, permissions...))

	var got []openapi.DeploymentConnectionSummary
	request := handler.Request{DeploymentId: deployment.ID, Limit: new(2)}
	for range 2 {
		response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
		require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
		got = append(got, response.Body.Data...)
		request.Cursor = response.Body.Pagination.Cursor
	}
	require.Nil(t, request.Cursor)
	require.ElementsMatch(t, want, got)
}

func TestListConnectionsIgnoresSnapshotsWithMismatchedCallerOrSettings(t *testing.T) {
	h, route, deployment := setup(t)
	for _, field := range []string{"workspace", "project", "app", "environment", "deployment"} {
		caller := deployment
		switch field {
		case "workspace":
			caller.WorkspaceID = uid.New(uid.WorkspacePrefix)
		case "project":
			caller.ProjectID = uid.New(uid.ProjectPrefix)
		case "app":
			caller.AppID = uid.New(uid.AppPrefix)
		case "environment":
			caller.EnvironmentID = uid.New(uid.EnvironmentPrefix)
		case "deployment":
			caller.ID = uid.New(uid.DeploymentPrefix)
		}
		seedSnapshot(t, h, caller, uid.New(uid.AppPrefix), "a-malformed-"+field, db.InsertDeploymentConnectionAppTargetParams{
			SelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic,
		})
	}
	id := uid.New(uid.ConnectionPrefix)
	insertSnapshot(t, h, deployment, id, uid.New(uid.AppPrefix), "b-missing-settings")
	require.NoError(t, db.Query.InsertDeploymentConnectionAppTarget(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionAppTargetParams{
		DeploymentID:  uid.New(uid.DeploymentPrefix),
		ConnectionID:  id,
		SelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic,
	}))
	headers := readHeaders(h, deployment)
	request := handler.Request{DeploymentId: deployment.ID}
	empty := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, empty.Status, "%s", empty.RawBody)
	require.NotNil(t, empty.Body.Data)
	require.Empty(t, empty.Body.Data)
	validTargetID := uid.New(uid.AppPrefix)
	validID := seedSnapshot(t, h, deployment, validTargetID, "z-valid", db.InsertDeploymentConnectionAppTargetParams{
		SelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic,
	})
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Equal(t, []openapi.DeploymentConnectionSummary{automaticSummary(validID, validTargetID)}, response.Body.Data)
}

func TestListConnectionsRequiresExactDeploymentReadAndMasksNotFound(t *testing.T) {
	h, route, deployment := setup(t)
	headers := authHeaders(h.CreateRootKey(deployment.WorkspaceID, "unkey:v1:"+deployment.WorkspaceID+":**#*"))
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{DeploymentId: uid.New(uid.DeploymentPrefix)})
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", missing.Body.Error.Type)
	other := deployment
	other.ID = uid.New(uid.DeploymentPrefix)
	for _, test := range []struct {
		name        string
		workspaceID string
		permissions []string
	}{
		{name: "no read", workspaceID: deployment.WorkspaceID},
		{name: "write only", workspaceID: deployment.WorkspaceID, permissions: []string{deploymentScope(deployment) + "#write"}},
		{name: "other deployment read", workspaceID: deployment.WorkspaceID, permissions: []string{deploymentScope(other) + "#read"}},
		{
			name:        "foreign workspace",
			workspaceID: h.Resources().RootWorkspace.ID,
			permissions: []string{"unkey:v1:" + h.Resources().RootWorkspace.ID + ":**#*"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(test.workspaceID, test.permissions...))
			response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{DeploymentId: deployment.ID})
			require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
			require.Equal(t, missing.Body.Error, response.Body.Error)
		})
	}
}

func TestListConnectionsAllowsSoftDeletedCallerParents(t *testing.T) {
	h, route, deployment := setup(t)
	targetID := uid.New(uid.AppPrefix)
	id := seedSnapshot(t, h, deployment, targetID, "saved", db.InsertDeploymentConnectionAppTargetParams{
		SelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic,
	})
	require.NoError(t, db.Query.MarkAppDeleted(t.Context(), h.DB.RW(), db.MarkAppDeletedParams{
		ID:        deployment.AppID,
		DeletedAt: sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: true},
	}))
	require.NoError(t, db.Query.MarkProjectDeleted(t.Context(), h.DB.RW(), db.MarkProjectDeletedParams{
		ID:        deployment.ProjectID,
		DeletedAt: sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: true},
	}))
	headers := readHeaders(h, deployment)
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{DeploymentId: deployment.ID})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Equal(t, []openapi.DeploymentConnectionSummary{automaticSummary(id, targetID)}, response.Body.Data)
}

func setup(t *testing.T) (*testutil.Harness, *handler.Handler, db.Deployment) {
	t.Helper()
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        "Project",
		Slug:        "project",
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: project.WorkspaceID,
		ProjectID:   project.ID,
		Name:        "Caller",
		Slug:        "caller",
	})
	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: app.WorkspaceID,
		ProjectID:   app.ProjectID,
		AppID:       app.ID,
		Slug:        "preview",
	})
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusFailed,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
	})
	require.Empty(t, deployment.Capabilities)
	return h, route, deployment
}

func seedSnapshot(t *testing.T, h *testutil.Harness, deployment db.Deployment, targetID, name string, target db.InsertDeploymentConnectionAppTargetParams) string {
	t.Helper()
	if target.ConnectionID == "" {
		target.ConnectionID = uid.New(uid.ConnectionPrefix)
	}
	target.DeploymentID = deployment.ID
	insertSnapshot(t, h, deployment, target.ConnectionID, targetID, name)
	require.NoError(t, db.Query.InsertDeploymentConnectionAppTarget(t.Context(), h.DB.RW(), target))
	return target.ConnectionID
}

func insertSnapshot(t *testing.T, h *testutil.Harness, deployment db.Deployment, id, targetID, name string) {
	t.Helper()
	require.NoError(t, db.Query.InsertDeploymentConnection(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionParams{
		DeploymentID:  deployment.ID,
		ConnectionID:  id,
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

func deploymentScope(deployment db.Deployment) string {
	return "unkey:v1:" + deployment.WorkspaceID + ":projects/" + deployment.ProjectID + "/apps/" + deployment.AppID + "/environments/" + deployment.EnvironmentID + "/deployments/" + deployment.ID
}

func readHeaders(h *testutil.Harness, deployment db.Deployment) http.Header {
	return authHeaders(h.CreateRootKey(deployment.WorkspaceID,
		deploymentScope(deployment)+"#read",
		"unkey:v1:"+deployment.WorkspaceID+":projects/"+deployment.ProjectID+"/apps/*#read",
	))
}

func authHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}

func automaticSummary(id, targetAppID string) openapi.DeploymentConnectionSummary {
	return openapi.DeploymentConnectionSummary{
		Id:                    id,
		TargetAppId:           targetAppID,
		TargetType:            openapi.ConnectionTargetTypeAutomatic,
		TargetAppName:         nullable.NewNullNullable[string](),
		TargetEnvironmentId:   nullable.NewNullNullable[string](),
		TargetEnvironmentSlug: nullable.NewNullNullable[string](),
		TargetDeploymentId:    nullable.NewNullNullable[string](),
	}
}
