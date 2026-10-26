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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_connections_list_targets"
)

func TestListTargetsPaginatesReadyRunningDeploymentsNewestFirst(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	var want []string
	for _, createdAt := range []int64{100, 300, 300, 200} {
		want = append(want, seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, createdAt, true).ID)
	}
	want = []string{want[2], want[1], want[3], want[0]}
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 400, false)
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateStopped, 400, true)
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusStopped, mysqltype.DeploymentsDesiredStateRunning, 400, true)
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusFailed, mysqltype.DeploymentsDesiredStateRunning, 400, true)
	seedDeployment(t, h, seedTarget(t, h, caller, "other"), mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 400, true)
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))

	var got []string
	request := handler.Request{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID, TargetAppId: target.AppID, Limit: new(3)}
	for range 2 {
		response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
		require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
		for _, deployment := range response.Body.Data.Deployments {
			require.Equal(t, target.ID, deployment.EnvironmentId)
			require.Equal(t, openapi.DeploymentStatusReady, deployment.Status)
			got = append(got, deployment.Id)
		}
		request.Cursor = response.Body.Pagination.Cursor
	}
	require.Nil(t, request.Cursor)
	require.Equal(t, want, got)
}

func TestListTargetsOmitsUnreadableDeployments(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	permissions := []string{environmentScope(caller) + "#read", appScope(target) + "#read"}
	var want []string
	for i := range int64(150) {
		deployment := seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 1000-i, true)
		if i%50 == 0 {
			permissions = append(permissions, environmentScope(target)+"/deployments/"+deployment.ID+"#read")
			want = append(want, deployment.ID)
		}
	}
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, permissions...))
	request := handler.Request{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID, TargetAppId: target.AppID, Limit: new(2)}

	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Equal(t, []string{want[0], want[1]}, deploymentIDs(first.Body.Data.Deployments))
	require.True(t, first.Body.Pagination.HasMore)
	require.Equal(t, want[2], *first.Body.Pagination.Cursor)

	request.Cursor = first.Body.Pagination.Cursor
	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, second.Status, "%s", second.RawBody)
	require.Equal(t, []string{want[2]}, deploymentIDs(second.Body.Data.Deployments))
	require.False(t, second.Body.Pagination.HasMore)
}

func TestListTargetsReturnsPinnedDeploymentOnFirstPage(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 300, true)
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 200, true)
	pinned := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   target.WorkspaceID,
		ProjectID:     target.ProjectID,
		AppID:         target.AppID,
		EnvironmentID: target.ID,
		Status:        mysqltype.DeploymentsStatusStopped,
		CreatedAt:     100,
		GitBranch:     "release",
		ImageResolved: "registry.example.com/backend:release",
	})
	seedPin(t, h, caller, pinned, "backend")
	otherCaller := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: caller.WorkspaceID,
		ProjectID:   caller.ProjectID,
		AppID:       caller.AppID,
		Slug:        "other",
	})
	seedPin(t, h, otherCaller, seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 50, true), "backend")
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))
	request := handler.Request{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID, TargetAppId: target.AppID, Limit: new(1)}

	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.NotNil(t, first.Body.Data.Pinned)
	require.Equal(t, pinned.ID, first.Body.Data.Pinned.Id)
	require.Equal(t, openapi.DeploymentStatusStopped, first.Body.Data.Pinned.Status)
	branch, err := first.Body.Data.Pinned.GitBranch.Get()
	require.NoError(t, err)
	require.Equal(t, "release", branch)
	image, err := first.Body.Data.Pinned.Image.Get()
	require.NoError(t, err)
	require.Equal(t, "registry.example.com/backend:release", image)
	require.True(t, first.Body.Pagination.HasMore)

	request.Cursor = first.Body.Pagination.Cursor
	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, second.Status, "%s", second.RawBody)
	require.Nil(t, second.Body.Data.Pinned)
	require.Len(t, second.Body.Data.Deployments, 1)

	headers = authHeaders(h.CreateRootKey(caller.WorkspaceID, environmentScope(caller)+"#read", appScope(target)+"#read"))
	request.Cursor = nil
	unreadable := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, unreadable.Status, "%s", unreadable.RawBody)
	require.Nil(t, unreadable.Body.Data.Pinned)
	require.Empty(t, unreadable.Body.Data.Deployments)
}

func TestListTargetsMasksUnreadableAndForeignCallerAndTargetScopes(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	otherProject := seedCaller(t, h, caller.WorkspaceID)
	foreign := seedCaller(t, h, h.CreateWorkspace().ID)
	for _, test := range []struct {
		name          string
		caller        db.Environment
		targetID      string
		permissions   []string
		targetMissing bool
	}{
		{name: "unreadable caller", caller: caller, targetID: target.AppID, permissions: []string{appScope(target) + "#read"}},
		{name: "write-only caller", caller: caller, targetID: target.AppID, permissions: []string{environmentScope(caller) + "#write", appScope(target) + "#read"}},
		{name: "foreign caller", caller: foreign, targetID: target.AppID, permissions: []string{"unkey:v1:" + caller.WorkspaceID + ":**#*"}},
		{name: "unreadable target", caller: caller, targetID: target.AppID, permissions: []string{environmentScope(caller) + "#read"}, targetMissing: true},
		{name: "other project target", caller: caller, targetID: otherProject.AppID, permissions: []string{"unkey:v1:" + caller.WorkspaceID + ":**#*"}, targetMissing: true},
		{name: "foreign target", caller: caller, targetID: foreign.AppID, permissions: []string{"unkey:v1:" + caller.WorkspaceID + ":**#*", appScope(foreign) + "#read"}, targetMissing: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, test.permissions...))
			request := handler.Request{Project: test.caller.ProjectID, App: test.caller.AppID, Environment: test.caller.ID, TargetAppId: test.targetID}
			denied := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			if test.targetMissing {
				request.TargetAppId = uid.New(uid.AppPrefix)
			} else {
				request.Environment = uid.New(uid.EnvironmentPrefix)
			}
			missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusNotFound, denied.Status, "%s", denied.RawBody)
			require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
			require.Equal(t, missing.Body.Error, denied.Body.Error)
		})
	}
}

func TestListTargetsRejectsCallerApp(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))
	request := handler.Request{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID, TargetAppId: caller.AppID}
	response := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusBadRequest, response.Status, "%s", response.RawBody)
}

func TestListTargetsIgnoresCursorFromAnotherApp(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 100, true)
	other := seedDeployment(t, h, seedTarget(t, h, caller, "other"), mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, 200, true)
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))
	for _, cursor := range []string{other.ID, uid.New(uid.DeploymentPrefix)} {
		response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project:     caller.ProjectID,
			App:         caller.AppID,
			Environment: caller.ID,
			TargetAppId: target.AppID,
			Cursor:      new(cursor),
		})
		require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
		require.Empty(t, response.Body.Data.Deployments)
		require.False(t, response.Body.Pagination.HasMore)
	}
}

func deploymentIDs(deployments []openapi.ConnectionTargetDeployment) []string {
	ids := make([]string, 0, len(deployments))
	for _, deployment := range deployments {
		ids = append(ids, deployment.Id)
	}
	return ids
}

func seedCaller(t *testing.T, h *testutil.Harness, workspaceID string) db.Environment {
	t.Helper()
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Project",
		Slug:        "project-" + uid.New(uid.TestPrefix),
	})
	return seedTarget(t, h, db.Environment{WorkspaceID: workspaceID, ProjectID: project.ID}, "caller")
}

func seedTarget(t *testing.T, h *testutil.Harness, caller db.Environment, slug string) db.Environment {
	t.Helper()
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: caller.WorkspaceID,
		ProjectID:   caller.ProjectID,
		Name:        slug,
		Slug:        slug,
	})
	return h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: caller.WorkspaceID,
		ProjectID:   caller.ProjectID,
		AppID:       app.ID,
		Slug:        "preview",
	})
}

func seedDeployment(t *testing.T, h *testutil.Harness, environment db.Environment, status mysqltype.DeploymentsStatus, desired mysqltype.DeploymentsDesiredState, createdAt int64, ready bool) db.Deployment {
	t.Helper()
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		Status:        status,
		DesiredState:  desired,
		CreatedAt:     createdAt,
	})
	if ready {
		require.NoError(t, db.Query.UpdateDeploymentFirstReadyAt(t.Context(), h.DB.RW(), db.UpdateDeploymentFirstReadyAtParams{
			ID:           deployment.ID,
			FirstReadyAt: sql.NullInt64{Int64: createdAt + 1, Valid: true},
		}))
	}
	return deployment
}

func seedPin(t *testing.T, h *testutil.Harness, caller db.Environment, deployment db.Deployment, name string) {
	t.Helper()
	id := uid.New(uid.ConnectionPrefix)
	require.NoError(t, db.Query.InsertAppConnection(t.Context(), h.DB.RW(), db.InsertAppConnectionParams{
		ID:            id,
		WorkspaceID:   caller.WorkspaceID,
		ProjectID:     caller.ProjectID,
		AppID:         caller.AppID,
		EnvironmentID: caller.ID,
		ResourceType:  db.AppConnectionsResourceTypeApp,
		ResourceID:    deployment.AppID,
		Name:          name,
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertConnectionAppTarget(t.Context(), h.DB.RW(), db.InsertConnectionAppTargetParams{
		ConnectionID:       id,
		SelectionMode:      db.ConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: deployment.ID, Valid: true},
	}))
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
