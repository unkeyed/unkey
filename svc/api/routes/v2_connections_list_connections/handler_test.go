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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_connections_list_connections"
)

func TestListConnectionsPaginatesEnvironmentConnections(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	environment := seedEnvironment(t, h)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))

	expected := make(map[string]openapi.AppConnection)
	var target db.App
	for _, mode := range []db.ConnectionAppTargetsSelectionMode{"automatic", "environment", "deployment"} {
		target = h.CreateApp(seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: environment.WorkspaceID,
			ProjectID:   environment.ProjectID,
			Name:        "Target " + string(mode),
			Slug:        "target-" + string(mode),
		})
		targetEnvironmentID := uid.New(uid.EnvironmentPrefix)
		targetDeploymentID := uid.New(uid.DeploymentPrefix)
		id := seedConnection(t, h, environment, target.ID, string(mode), db.InsertConnectionAppTargetParams{
			SelectionMode:       mode,
			TargetEnvironmentID: sql.NullString{String: targetEnvironmentID, Valid: mode == "environment"},
			TargetDeploymentID:  sql.NullString{String: targetDeploymentID, Valid: mode == "deployment"},
		})
		row := openapi.AppConnection{
			Id:            id,
			AppId:         environment.AppID,
			EnvironmentId: environment.ID,
			Name:          string(mode),
			TargetAppId:   target.ID,
			TargetAppName: target.Name,
			TargetAppSlug: target.Slug,
			TargetType:    openapi.ConnectionTargetType(mode),
		}
		if mode == "environment" {
			row.TargetEnvironmentId = new(targetEnvironmentID)
		}
		if mode == "deployment" {
			row.TargetDeploymentId = new(targetDeploymentID)
		}
		expected[id] = row
	}
	seedConnection(t, h, environment, environment.AppID, "self", db.InsertConnectionAppTargetParams{SelectionMode: "automatic"})
	other := environment
	other.ID = uid.New(uid.EnvironmentPrefix)
	seedConnection(t, h, other, target.ID, "other-environment", db.InsertConnectionAppTargetParams{SelectionMode: "automatic"})

	request := handler.Request{Project: environment.ProjectID, App: environment.AppID, Environment: environment.ID, Limit: new(2)}
	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Len(t, first.Body.Data, 2)
	require.True(t, first.Body.Pagination.HasMore)
	require.NotNil(t, first.Body.Pagination.Cursor)
	request.Cursor = first.Body.Pagination.Cursor
	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, second.Status, "%s", second.RawBody)
	require.Len(t, second.Body.Data, 1)
	require.False(t, second.Body.Pagination.HasMore)
	require.Nil(t, second.Body.Pagination.Cursor)
	require.Equal(t, *first.Body.Pagination.Cursor, second.Body.Data[0].Id)

	for _, row := range append(first.Body.Data, second.Body.Data...) {
		require.Contains(t, expected, row.Id)
		require.Equal(t, expected[row.Id], row)
		delete(expected, row.Id)
	}
	require.Empty(t, expected)
}

func TestListConnectionsDoesNotRevealUnreadableEnvironment(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	environment := seedEnvironment(t, h)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID))
	request := handler.Request{Project: environment.ProjectID, App: environment.AppID, Environment: environment.ID}
	denied := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	request.Environment = uid.New(uid.EnvironmentPrefix)
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusNotFound, denied.Status, "%s", denied.RawBody)
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, missing.Body.Error, denied.Body.Error)

	request.Environment = environment.ID
	headers = authHeaders(h.CreateRootKey(h.Resources().RootWorkspace.ID, "unkey:v1:"+h.Resources().RootWorkspace.ID+":**#*"))
	foreign := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusNotFound, foreign.Status, "%s", foreign.RawBody)
	require.Equal(t, missing.Body.Error, foreign.Body.Error)
}

func TestListConnectionsOmitsUnreadableTargetApps(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	environment := seedEnvironment(t, h)
	var readable, hidden db.App
	for _, slug := range []string{"readable", "hidden"} {
		target := h.CreateApp(seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: environment.WorkspaceID,
			ProjectID:   environment.ProjectID,
			Name:        slug,
			Slug:        slug,
		})
		seedConnection(t, h, environment, target.ID, slug, db.InsertConnectionAppTargetParams{SelectionMode: db.ConnectionAppTargetsSelectionModeAutomatic})
		if slug == "readable" {
			readable = target
		} else {
			hidden = target
		}
	}
	projectScope := "unkey:v1:" + environment.WorkspaceID + ":projects/" + environment.ProjectID
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID,
		projectScope+"/apps/"+environment.AppID+"/environments/"+environment.ID+"#read",
		projectScope+"/apps/"+readable.ID+"#read",
	))
	request := handler.Request{Project: "project", App: "caller", Environment: "preview", Limit: new(1)}
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Len(t, response.Body.Data, 1)
	require.Equal(t, readable.ID, response.Body.Data[0].TargetAppId)
	require.NotEqual(t, hidden.ID, response.Body.Data[0].TargetAppId)
	require.False(t, response.Body.Pagination.HasMore)
	require.Nil(t, response.Body.Pagination.Cursor)

	request.Limit = new(101)
	invalid := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusBadRequest, invalid.Status, "%s", invalid.RawBody)
	request.Limit = nil
	unauthenticated := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, authHeaders("invalid-token"), request)
	require.Equal(t, http.StatusUnauthorized, unauthenticated.Status, "%s", unauthenticated.RawBody)
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

func seedConnection(t *testing.T, h *testutil.Harness, environment db.Environment, targetID, name string, selection db.InsertConnectionAppTargetParams) string {
	t.Helper()
	id := uid.New(uid.ConnectionPrefix)
	require.NoError(t, db.Query.InsertAppConnection(t.Context(), h.DB.RW(), db.InsertAppConnectionParams{
		ID:            id,
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		ResourceType:  "app",
		ResourceID:    targetID,
		Name:          name,
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	selection.ConnectionID = id
	require.NoError(t, db.Query.InsertConnectionAppTarget(t.Context(), h.DB.RW(), selection))
	return id
}

func authHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}
