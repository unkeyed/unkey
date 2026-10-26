package handler_test

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_connections_create_connection"
)

func TestCreateConnectionPersistsSelectionWithExactPermissions(t *testing.T) {
	for _, test := range []struct {
		name       string
		mode       openapi.ConnectionTargetType
		slugs      bool
		nameCustom *string
	}{
		{name: "automatic by id", mode: openapi.ConnectionTargetTypeAutomatic, nameCustom: new("backend")},
		{name: "automatic by slug with derived name", mode: openapi.ConnectionTargetTypeAutomatic, slugs: true},
		{name: "environment with longest valid name", mode: openapi.ConnectionTargetTypeEnvironment, slugs: true, nameCustom: new(strings.Repeat("a", 63))},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
			h.Register(route)
			require.Equal(t, http.MethodPost, route.Method())
			require.Equal(t, "/v2/connections.createConnection", route.Path())
			environment := seedEnvironment(t, h)
			target := seedTarget(t, h, environment, "target")
			request := connectionRequest(environment, target)
			request.Name = test.nameCustom
			request.TargetType = test.mode
			request.TargetEnvironmentId = new(uid.New(uid.EnvironmentPrefix))
			request.TargetDeploymentId = new(uid.New(uid.DeploymentPrefix))
			if test.slugs {
				request.Project = "project"
				request.App = "caller"
				request.Environment = environment.Slug
			}
			permissions := []string{environmentScope(environment) + "#read", environmentScope(environment) + "#write", appScope(target) + "#read"}
			var targetEnvironmentID sql.NullString
			if test.mode == openapi.ConnectionTargetTypeEnvironment {
				targetEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
					ID:          uid.New(uid.EnvironmentPrefix),
					WorkspaceID: target.WorkspaceID,
					ProjectID:   target.ProjectID,
					AppID:       target.ID,
					Slug:        "production",
				})
				request.TargetEnvironmentId = new(targetEnvironment.ID)
				targetEnvironmentID = sql.NullString{String: targetEnvironment.ID, Valid: true}
				permissions = append(permissions, environmentScope(targetEnvironment)+"#read")
			}

			headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, permissions...))
			response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
			require.Equal(t, http.StatusCreated, response.Status, "%s", response.RawBody)
			require.NotEmpty(t, response.Body.Data.Id)
			require.NotEmpty(t, response.Body.Meta.RequestId)
			name := target.Slug
			if test.nameCustom != nil {
				name = *test.nameCustom
			}
			require.Equal(t, name, response.Body.Data.Name)
			require.Equal(t, []db.ListAppConnectionsRow{{
				ID:            response.Body.Data.Id,
				AppID:         environment.AppID,
				EnvironmentID: environment.ID,
				TargetAppID:   target.ID,
				TargetAppName: target.Name,
				TargetAppSlug: target.Slug,
				Name:          name,
				SelectionMode: db.NullConnectionAppTargetsSelectionMode{
					ConnectionAppTargetsSelectionMode: db.ConnectionAppTargetsSelectionMode(test.mode),
					Valid:                             true,
				},
				TargetEnvironmentID: targetEnvironmentID,
			}}, listConnections(t, h, environment))
		})
	}
}

func TestCreateConnectionPinsOnlyPreviouslyReadyRunningDeployments(t *testing.T) {
	for _, test := range []struct {
		name                 string
		readyPreviously      bool
		status               mysqltype.DeploymentsStatus
		desiredState         mysqltype.DeploymentsDesiredState
		statusHTTP           int
		appUnreadable        bool
		deploymentUnreadable bool
	}{
		{name: "ready and running", readyPreviously: true, status: mysqltype.DeploymentsStatusReady, desiredState: mysqltype.DeploymentsDesiredStateRunning, statusHTTP: http.StatusCreated},
		{name: "never ready", status: mysqltype.DeploymentsStatusReady, desiredState: mysqltype.DeploymentsDesiredStateRunning, statusHTTP: http.StatusConflict},
		{name: "status stopped", readyPreviously: true, status: mysqltype.DeploymentsStatusStopped, desiredState: mysqltype.DeploymentsDesiredStateRunning, statusHTTP: http.StatusConflict},
		{name: "desired stopped", readyPreviously: true, status: mysqltype.DeploymentsStatusReady, desiredState: mysqltype.DeploymentsDesiredStateStopped, statusHTTP: http.StatusConflict},
		{name: "unreadable deployment", readyPreviously: true, status: mysqltype.DeploymentsStatusReady, desiredState: mysqltype.DeploymentsDesiredStateRunning, statusHTTP: http.StatusNotFound, deploymentUnreadable: true},
		{name: "unreadable app", readyPreviously: true, status: mysqltype.DeploymentsStatusReady, desiredState: mysqltype.DeploymentsDesiredStateRunning, statusHTTP: http.StatusNotFound, appUnreadable: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
			h.Register(route)
			environment := seedEnvironment(t, h)
			target := seedTarget(t, h, environment, "target")
			targetEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
				ID:          uid.New(uid.EnvironmentPrefix),
				WorkspaceID: target.WorkspaceID,
				ProjectID:   target.ProjectID,
				AppID:       target.ID,
				Slug:        "production",
			})
			deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
				ID:            uid.New(uid.DeploymentPrefix),
				WorkspaceID:   target.WorkspaceID,
				ProjectID:     target.ProjectID,
				AppID:         target.ID,
				EnvironmentID: targetEnvironment.ID,
				Status:        test.status,
				DesiredState:  test.desiredState,
			})
			require.NoError(t, db.Query.UpdateDeploymentFirstReadyAt(t.Context(), h.DB.RW(), db.UpdateDeploymentFirstReadyAtParams{
				ID:           deployment.ID,
				FirstReadyAt: sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: test.readyPreviously},
			}))
			permissions := []string{environmentScope(environment) + "#read", environmentScope(environment) + "#write"}
			if !test.appUnreadable {
				permissions = append(permissions, appScope(target)+"#read")
			}
			if !test.deploymentUnreadable {
				permissions = append(permissions, environmentScope(targetEnvironment)+"/deployments/"+deployment.ID+"#read")
			}
			headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, permissions...))
			request := connectionRequest(environment, target)
			request.Name = new("backend")
			request.TargetType = openapi.ConnectionTargetTypeDeployment
			request.TargetDeploymentId = new(deployment.ID)
			request.TargetEnvironmentId = new(uid.New(uid.EnvironmentPrefix))

			if test.statusHTTP != http.StatusCreated {
				response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
				require.Equal(t, test.statusHTTP, response.Status, "%s", response.RawBody)
				if test.statusHTTP == http.StatusNotFound {
					if test.appUnreadable {
						request.TargetApp = uid.New(uid.AppPrefix)
					} else {
						request.TargetDeploymentId = new(uid.New(uid.DeploymentPrefix))
					}
					missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
					require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
					require.Equal(t, missing.Body.Error, response.Body.Error)
				}
				require.Empty(t, listConnections(t, h, environment))
				return
			}

			response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
			require.Equal(t, http.StatusCreated, response.Status, "%s", response.RawBody)
			require.NotEmpty(t, response.Body.Data.Id)
			require.Equal(t, "backend", response.Body.Data.Name)
			rows := listConnections(t, h, environment)
			require.Len(t, rows, 1)
			require.Equal(t, response.Body.Data.Id, rows[0].ID)
			require.Equal(t, response.Body.Data.Name, rows[0].Name)
			require.Equal(t, target.ID, rows[0].TargetAppID)
			require.True(t, rows[0].SelectionMode.Valid)
			require.Equal(t, db.ConnectionAppTargetsSelectionModeDeployment, rows[0].SelectionMode.ConnectionAppTargetsSelectionMode)
			require.Equal(t, sql.NullString{String: deployment.ID, Valid: true}, rows[0].TargetDeploymentID)
			require.False(t, rows[0].TargetEnvironmentID.Valid)
		})
	}
}

func TestCreateConnectionRequiresAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	for _, test := range []struct {
		name    string
		headers http.Header
		status  int
	}{
		{name: "missing bearer", headers: http.Header{"Content-Type": {"application/json"}}, status: http.StatusBadRequest},
		{name: "invalid bearer", headers: authHeaders("invalid-token"), status: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, test.headers, connectionRequest(environment, target))
			require.Equal(t, test.status, response.Status, "%s", response.RawBody)
			require.Empty(t, listConnections(t, h, environment))
		})
	}
}

func TestCreateConnectionRequiresCallerReadAndWrite(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	for _, test := range []struct {
		name        string
		permissions []string
		status      int
	}{
		{name: "no caller access", status: http.StatusNotFound},
		{name: "write without read", permissions: []string{environmentScope(environment) + "#write"}, status: http.StatusNotFound},
		{name: "read without write", permissions: []string{environmentScope(environment) + "#read"}, status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			permissions := append([]string{appScope(target) + "#read"}, test.permissions...)
			headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, permissions...))
			request := connectionRequest(environment, target)
			denied := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, test.status, denied.Status, "%s", denied.RawBody)
			if test.status == http.StatusNotFound {
				request.Environment = uid.New(uid.EnvironmentPrefix)
				missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
				require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
				require.Equal(t, missing.Body.Error, denied.Body.Error)
			}
			require.Empty(t, listConnections(t, h, environment))
		})
	}
}

func TestCreateConnectionDoesNotRevealUnreadableTargets(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	targetEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: target.WorkspaceID,
		ProjectID:   target.ProjectID,
		AppID:       target.ID,
		Slug:        "production",
	})
	for _, test := range []struct {
		name              string
		mode              openapi.ConnectionTargetType
		permissions       []string
		environmentDenied bool
	}{
		{name: "automatic target app", mode: openapi.ConnectionTargetTypeAutomatic},
		{name: "environment read does not grant app read", mode: openapi.ConnectionTargetTypeEnvironment, permissions: []string{environmentScope(targetEnvironment) + "#read"}},
		{name: "app read does not grant environment read", mode: openapi.ConnectionTargetTypeEnvironment, permissions: []string{appScope(target) + "#read"}, environmentDenied: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			permissions := append([]string{environmentScope(environment) + "#read", environmentScope(environment) + "#write"}, test.permissions...)
			headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, permissions...))
			request := connectionRequest(environment, target)
			request.TargetType = test.mode
			if test.mode == openapi.ConnectionTargetTypeEnvironment {
				request.TargetEnvironmentId = new(targetEnvironment.ID)
			}
			denied := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusNotFound, denied.Status, "%s", denied.RawBody)
			if test.environmentDenied {
				request.TargetEnvironmentId = new(uid.New(uid.EnvironmentPrefix))
			} else {
				request.TargetApp = uid.New(uid.AppPrefix)
			}
			missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
			require.Equal(t, missing.Body.Error, denied.Body.Error)
			require.Empty(t, listConnections(t, h, environment))
		})
	}
}

func TestCreateConnectionDoesNotRevealForeignCaller(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	workspace := h.Resources().RootWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":**#*"))
	request := connectionRequest(environment, target)
	foreign := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	request.Environment = uid.New(uid.EnvironmentPrefix)
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusNotFound, foreign.Status, "%s", foreign.RawBody)
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, missing.Body.Error, foreign.Body.Error)
	require.Empty(t, listConnections(t, h, environment))
}

func TestCreateConnectionRejectsTargetsOutsideCallerScope(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	for _, test := range []struct {
		name        string
		workspaceID string
	}{
		{name: "another project", workspaceID: environment.WorkspaceID},
		{name: "another workspace", workspaceID: h.Resources().RootWorkspace.ID},
	} {
		t.Run(test.name, func(t *testing.T) {
			project := h.CreateProject(seed.CreateProjectRequest{
				ID:          uid.New(uid.ProjectPrefix),
				WorkspaceID: test.workspaceID,
				Name:        "Other project",
				Slug:        "other-project",
			})
			targetOther := h.CreateApp(seed.CreateAppRequest{
				ID:          uid.New(uid.AppPrefix),
				WorkspaceID: project.WorkspaceID,
				ProjectID:   project.ID,
				Name:        "Other target",
				Slug:        "other-target",
			})
			headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*", appScope(targetOther)+"#read"))
			request := connectionRequest(environment, targetOther)
			denied := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			request.TargetApp = uid.New(uid.AppPrefix)
			missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusNotFound, denied.Status, "%s", denied.RawBody)
			require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
			require.Equal(t, missing.Body.Error, denied.Body.Error)

			environmentOther := h.CreateEnvironment(seed.CreateEnvironmentRequest{
				ID:          uid.New(uid.EnvironmentPrefix),
				WorkspaceID: targetOther.WorkspaceID,
				ProjectID:   targetOther.ProjectID,
				AppID:       targetOther.ID,
				Slug:        "production",
			})
			headers = authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*", environmentScope(environmentOther)+"#read"))
			request = connectionRequest(environment, target)
			request.TargetType = openapi.ConnectionTargetTypeEnvironment
			request.TargetEnvironmentId = new(environmentOther.ID)
			denied = testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			request.TargetEnvironmentId = new(uid.New(uid.EnvironmentPrefix))
			missing = testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusNotFound, denied.Status, "%s", denied.RawBody)
			require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
			require.Equal(t, missing.Body.Error, denied.Body.Error)
			require.Empty(t, listConnections(t, h, environment))
		})
	}

	request := connectionRequest(environment, target)
	request.TargetType = openapi.ConnectionTargetTypeEnvironment
	request.TargetEnvironmentId = new(environment.ID)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
	require.Empty(t, listConnections(t, h, environment))
}

func TestCreateConnectionRejectsSelfTarget(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	request := handler.Request{
		Project:     environment.ProjectID,
		App:         environment.AppID,
		Environment: environment.ID,
		TargetApp:   environment.AppID,
		TargetType:  openapi.ConnectionTargetTypeAutomatic,
		Name:        new("backend"),
	}
	response := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusBadRequest, response.Status, "%s", response.RawBody)
	require.Empty(t, listConnections(t, h, environment))
}

func TestCreateConnectionRejectsInvalidNames(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	for _, name := range []string{"", strings.Repeat("a", 64), "1backend", "Backend", "back_end", "back.end", "-backend", "backend-", "self", "unkey", "unkey-service", "unkeyboard"} {
		t.Run(name, func(t *testing.T) {
			request := connectionRequest(environment, target)
			request.Name = new(name)
			response := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusBadRequest, response.Status, "%s", response.RawBody)
			require.Empty(t, listConnections(t, h, environment))
		})
	}
}

func TestCreateConnectionRequiresSelectedTarget(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	for _, mode := range []openapi.ConnectionTargetType{openapi.ConnectionTargetTypeEnvironment, openapi.ConnectionTargetTypeDeployment, "unknown"} {
		t.Run(string(mode), func(t *testing.T) {
			request := connectionRequest(environment, target)
			request.TargetType = mode
			response := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusBadRequest, response.Status, "%s", response.RawBody)
			require.Empty(t, listConnections(t, h, environment))
		})
	}
}

func TestCreateConnectionRejectsDuplicateNameOrTargetWithinEnvironment(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	targetOther := seedTarget(t, h, environment, "other-target")
	seedConnection(t, h, environment, target.ID, "backend")
	before := listConnections(t, h, environment)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	for _, test := range []struct {
		name   string
		target db.App
		alias  string
	}{
		{name: "duplicate target with another name", target: target, alias: "another-name"},
		{name: "duplicate name with another target", target: targetOther, alias: "backend"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := connectionRequest(environment, test.target)
			request.Name = new(test.alias)
			response := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusConflict, response.Status, "%s", response.RawBody)
			require.Equal(t, before, listConnections(t, h, environment))
		})
	}

	environmentOther := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		AppID:       environment.AppID,
		Slug:        "another-preview",
	})
	request := connectionRequest(environmentOther, target)
	request.Name = new("backend")
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusCreated, response.Status, "%s", response.RawBody)
	require.Equal(t, "backend", response.Body.Data.Name)
	rows := listConnections(t, h, environmentOther)
	require.Len(t, rows, 1)
	require.Equal(t, response.Body.Data.Id, rows[0].ID)
	require.Equal(t, target.ID, rows[0].TargetAppID)
	require.Equal(t, before, listConnections(t, h, environment))
}

func TestCreateConnectionDerivesUniqueNameOnCollision(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)
	environment := seedEnvironment(t, h)
	target := seedTarget(t, h, environment, "target")
	targetOther := seedTarget(t, h, environment, "other-target")
	seedConnection(t, h, environment, targetOther.ID, target.Slug)
	headers := authHeaders(h.CreateRootKey(environment.WorkspaceID, "unkey:v1:"+environment.WorkspaceID+":**#*"))
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, connectionRequest(environment, target))
	require.Equal(t, http.StatusCreated, response.Status, "%s", response.RawBody)
	require.NotEqual(t, target.Slug, response.Body.Data.Name)
	require.True(t, strings.HasPrefix(response.Body.Data.Name, target.Slug))
	require.Regexp(t, `^[a-z]([a-z0-9-]*[a-z0-9])?$`, response.Body.Data.Name)
	require.LessOrEqual(t, len(response.Body.Data.Name), 63)
	rows := listConnections(t, h, environment)
	require.Len(t, rows, 2)
	for _, row := range rows {
		if row.TargetAppID == target.ID {
			require.Equal(t, response.Body.Data.Id, row.ID)
			require.Equal(t, response.Body.Data.Name, row.Name)
		} else {
			require.Equal(t, targetOther.ID, row.TargetAppID)
			require.Equal(t, target.Slug, row.Name)
		}
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

func seedTarget(t *testing.T, h *testutil.Harness, environment db.Environment, slug string) db.App {
	t.Helper()
	return h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: environment.WorkspaceID,
		ProjectID:   environment.ProjectID,
		Name:        "Target " + slug,
		Slug:        slug,
	})
}

func seedConnection(t *testing.T, h *testutil.Harness, environment db.Environment, targetID, name string) {
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
	require.NoError(t, db.Query.InsertConnectionAppTarget(t.Context(), h.DB.RW(), db.InsertConnectionAppTargetParams{
		ConnectionID:  id,
		SelectionMode: "automatic",
	}))
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

func connectionRequest(environment db.Environment, target db.App) handler.Request {
	return handler.Request{
		Project:     environment.ProjectID,
		App:         environment.AppID,
		Environment: environment.ID,
		TargetApp:   target.ID,
		TargetType:  openapi.ConnectionTargetTypeAutomatic,
	}
}

func environmentScope(environment db.Environment) string {
	return "unkey:v1:" + environment.WorkspaceID + ":projects/" + environment.ProjectID + "/apps/" + environment.AppID + "/environments/" + environment.ID
}

func appScope(app db.App) string {
	return "unkey:v1:" + app.WorkspaceID + ":projects/" + app.ProjectID + "/apps/" + app.ID
}

func authHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}
