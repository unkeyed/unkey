package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"strings"
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

func TestListTargetsPaginatesTiedTimesWithInclusiveCursorAndRepeatedDeduplicatedPins(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	targetOld := seedTarget(t, h, caller, "old-target")
	pinOld := seedDeployment(t, h, targetOld, mysqltype.DeploymentsStatusFailed, 1, true)
	seedPin(t, h, caller, pinOld, "old-pin")
	ids := make([]string, 0, 101)
	var choice db.Deployment
	for range 101 {
		choice = seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, 100, true)
		ids = append(ids, choice.ID)
	}
	slices.SortFunc(ids, func(a, b string) int { return strings.Compare(strings.ToLower(b), strings.ToLower(a)) })
	seedPin(t, h, caller, choice, "choice-pin")
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))
	request := handler.Request{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID, TargetAppId: new(target.AppID)}
	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.True(t, first.Body.Pagination.HasMore)
	require.NotNil(t, first.Body.Pagination.Cursor)
	require.Equal(t, ids[100], *first.Body.Pagination.Cursor)
	require.NotEmpty(t, first.Body.Meta.RequestId)

	request.Cursor = first.Body.Pagination.Cursor
	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, second.Status, "%s", second.RawBody)
	require.False(t, second.Body.Pagination.HasMore)
	require.Nil(t, second.Body.Pagination.Cursor)
	for _, page := range []struct {
		response *handler.Response
		choices  []string
	}{
		{response: first.Body, choices: ids[:100]},
		{response: second.Body, choices: ids[100:]},
	} {
		rows := page.response.Data.Deployments
		require.GreaterOrEqual(t, len(rows), 2)
		require.ElementsMatch(t, []string{pinOld.ID, choice.ID}, []string{rows[0].Id, rows[1].Id})
		expected := []string{pinOld.ID, choice.ID}
		for _, id := range page.choices {
			if id != choice.ID {
				expected = append(expected, id)
			}
		}
		require.Len(t, rows, len(expected))
		for i, row := range rows[2:] {
			require.Equal(t, expected[i+2], row.Id)
		}
		seen := make(map[string]bool)
		for _, row := range rows {
			require.False(t, seen[row.Id], "duplicate deployment %s", row.Id)
			seen[row.Id] = true
		}
		require.True(t, seen[pinOld.ID])
		for _, id := range page.choices {
			require.True(t, seen[id], "missing choice %s", id)
		}
	}
}

func TestListTargetsScansDeniedBatchesAndUsesOnlyAuthorizedLookahead(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	for range 102 {
		seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, 300, true)
	}
	permissions := []string{environmentScope(caller) + "#read", appScope(target) + "#read"}
	ids := make([]string, 0, 101)
	for range 101 {
		deployment := seedDeployment(t, h, target, mysqltype.DeploymentsStatusStopped, 200, true)
		permissions = append(permissions, environmentScope(target)+"/deployments/"+deployment.ID+"#read")
		ids = append(ids, deployment.ID)
	}
	slices.SortFunc(ids, func(a, b string) int { return strings.Compare(strings.ToLower(b), strings.ToLower(a)) })
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, 100, true)
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, permissions...))
	request := handler.Request{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID, TargetAppId: new(target.AppID)}
	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Len(t, first.Body.Data.Deployments, 100)
	for i, row := range first.Body.Data.Deployments {
		require.Equal(t, ids[i], row.Id)
		require.Equal(t, openapi.DeploymentStatusStopped, row.Status)
	}
	require.True(t, first.Body.Pagination.HasMore)
	require.NotNil(t, first.Body.Pagination.Cursor)
	require.Equal(t, ids[100], *first.Body.Pagination.Cursor)
	request.Cursor = first.Body.Pagination.Cursor
	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, second.Status, "%s", second.RawBody)
	require.Len(t, second.Body.Data.Deployments, 1)
	require.Equal(t, ids[100], second.Body.Data.Deployments[0].Id)
	require.False(t, second.Body.Pagination.HasMore)
	require.Nil(t, second.Body.Pagination.Cursor)
	require.Equal(t, []openapi.ConnectionTargetEnvironment{{Id: caller.ID, AppId: caller.AppID, Slug: caller.Slug, Kind: openapi.ConnectionTargetEnvironmentKind(caller.Kind)}}, first.Body.Data.Environments)
}

func TestListTargetsPinsOnlyIncludesPreviouslyReadyPinsAndNullableMetadata(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	permissions := []string{environmentScope(caller) + "#read"}
	var expected []string
	for _, test := range []struct {
		slug   string
		status mysqltype.DeploymentsStatus
		ready  bool
		read   bool
	}{
		{slug: "failed-pin", status: mysqltype.DeploymentsStatusFailed, ready: true, read: true},
		{slug: "stopped-pin", status: mysqltype.DeploymentsStatusStopped, ready: true, read: true},
		{slug: "never-ready-pin", status: mysqltype.DeploymentsStatusReady, read: true},
		{slug: "unreadable-pin", status: mysqltype.DeploymentsStatusReady, ready: true},
	} {
		target := seedTarget(t, h, caller, test.slug)
		deployment := seedDeployment(t, h, target, test.status, 100, test.ready)
		seedPin(t, h, caller, deployment, test.slug)
		if test.read {
			permissions = append(permissions, environmentScope(target)+"/deployments/"+deployment.ID+"#read")
			if test.ready {
				expected = append(expected, deployment.ID)
			}
		}
	}
	unpinned := seedTarget(t, h, caller, "unpinned")
	seedDeployment(t, h, unpinned, mysqltype.DeploymentsStatusReady, 200, true)
	permissions = append(permissions, appScope(unpinned)+"/**#read", appScope(unpinned)+"#read")
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, permissions...))
	project, err := db.Query.FindProjectById(t.Context(), h.DB.RO(), caller.ProjectID)
	require.NoError(t, err)
	for _, selectors := range []handler.Request{
		{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID},
		{Project: project.Slug, App: "caller", Environment: caller.Slug},
	} {
		response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, selectors)
		require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
		require.False(t, response.Body.Pagination.HasMore)
		require.Nil(t, response.Body.Pagination.Cursor)
		var ids []string
		for _, row := range response.Body.Data.Deployments {
			ids = append(ids, row.Id)
			require.True(t, row.GitBranch.IsNull())
			require.True(t, row.Image.IsNull())
		}
		require.ElementsMatch(t, expected, ids)
		require.Equal(t, []openapi.ConnectionTargetApp{{Id: unpinned.AppID, Name: "unpinned", Slug: "unpinned"}}, response.Body.Data.Apps)
	}
}

func TestListTargetsChoicesExcludeNeverReadyAndFailedDeploymentsAndPreserveMetadata(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, 500, false)
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusStopped, 400, false)
	seedDeployment(t, h, target, mysqltype.DeploymentsStatusFailed, 300, true)
	stopped := seedDeployment(t, h, target, mysqltype.DeploymentsStatusStopped, 200, true)
	ready := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   target.WorkspaceID,
		ProjectID:     target.ProjectID,
		AppID:         target.AppID,
		EnvironmentID: target.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     100,
		GitBranch:     "release",
		ImageResolved: "registry.example.com/backend:release",
		Capabilities:  mysqltype.DeploymentCapabilities{PrivateNetworking: true},
	})
	require.NoError(t, db.Query.UpdateDeploymentFirstReadyAt(t.Context(), h.DB.RW(), db.UpdateDeploymentFirstReadyAtParams{
		ID:           ready.ID,
		FirstReadyAt: sql.NullInt64{Int64: 101, Valid: true},
	}))
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project:     caller.ProjectID,
		App:         caller.AppID,
		Environment: caller.ID,
		TargetAppId: new(target.AppID),
	})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Len(t, response.Body.Data.Deployments, 2)
	require.Equal(t, stopped.ID, response.Body.Data.Deployments[0].Id)
	row := response.Body.Data.Deployments[1]
	require.Equal(t, ready.ID, row.Id)
	require.Equal(t, target.AppID, row.AppId)
	require.Equal(t, target.ID, row.EnvironmentId)
	branch, err := row.GitBranch.Get()
	require.NoError(t, err)
	require.Equal(t, "release", branch)
	image, err := row.Image.Get()
	require.NoError(t, err)
	require.Equal(t, ready.ImageResolved.String, image)
	require.False(t, response.Body.Pagination.HasMore)
	require.Nil(t, response.Body.Pagination.Cursor)
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
		name        string
		caller      db.Environment
		targetID    *string
		permissions []string
	}{
		{name: "unreadable caller", caller: caller},
		{name: "write-only caller", caller: caller, permissions: []string{environmentScope(caller) + "#write"}},
		{name: "foreign caller", caller: foreign, permissions: []string{"unkey:v1:" + caller.WorkspaceID + ":**#*"}},
		{name: "unreadable target", caller: caller, targetID: new(target.AppID), permissions: []string{environmentScope(caller) + "#read"}},
		{name: "other project target", caller: caller, targetID: new(otherProject.AppID), permissions: []string{"unkey:v1:" + caller.WorkspaceID + ":**#*"}},
		{name: "foreign target", caller: caller, targetID: new(foreign.AppID), permissions: []string{"unkey:v1:" + caller.WorkspaceID + ":**#*", appScope(foreign) + "#read"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, test.permissions...))
			request := handler.Request{Project: test.caller.ProjectID, App: test.caller.AppID, Environment: test.caller.ID, TargetAppId: test.targetID}
			denied := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			if test.targetID == nil {
				request.Environment = uid.New(uid.EnvironmentPrefix)
			} else {
				request.TargetAppId = new(uid.New(uid.AppPrefix))
			}
			missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
			require.Equal(t, http.StatusNotFound, denied.Status, "%s", denied.RawBody)
			require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
			require.Equal(t, missing.Body.Error, denied.Body.Error)
		})
	}
}

func TestListTargetsMasksGuessedCursorAndRejectsCursorWithoutTarget(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	otherApp := seedTarget(t, h, caller, "other")
	otherProject := seedCaller(t, h, caller.WorkspaceID)
	foreign := seedCaller(t, h, h.CreateWorkspace().ID)
	permissions := []string{environmentScope(caller) + "#read", appScope(target) + "#read"}
	unreadable := seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, 100, true)
	cursors := []string{unreadable.ID}
	for _, environment := range []db.Environment{otherApp, otherProject, foreign} {
		deployment := seedDeployment(t, h, environment, mysqltype.DeploymentsStatusReady, 100, true)
		cursors = append(cursors, deployment.ID)
		permissions = append(permissions, environmentScope(environment)+"/deployments/"+deployment.ID+"#read")
	}
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, permissions...))
	request := handler.Request{
		Project:     caller.ProjectID,
		App:         caller.AppID,
		Environment: caller.ID,
		TargetAppId: new(target.AppID),
		Cursor:      new(uid.New(uid.DeploymentPrefix)),
	}
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", missing.Body.Error.Type)
	for _, cursor := range cursors {
		request.Cursor = new(cursor)
		response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request)
		require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
		require.Equal(t, missing.Body.Error, response.Body.Error)
	}
	request.TargetAppId = nil
	invalid := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusBadRequest, invalid.Status, "%s", invalid.RawBody)
	for _, targetID := range []*string{nil, new(target.AppID)} {
		request.TargetAppId = targetID
		request.Cursor = new("")
		response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
		require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
		require.Empty(t, response.Body.Data.Deployments)
		require.False(t, response.Body.Pagination.HasMore)
		require.Nil(t, response.Body.Pagination.Cursor)
	}
}

func TestListTargetsCapsAppAndEnvironmentCandidatesBeforeAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	var apps []openapi.ConnectionTargetApp
	var target db.Environment
	for i := range 101 {
		slug := fmt.Sprintf("target-%03d", i)
		target = seedTarget(t, h, caller, slug)
		apps = append(apps, openapi.ConnectionTargetApp{Id: target.AppID, Name: slug, Slug: slug})
	}
	var environments []openapi.ConnectionTargetEnvironment
	var environment db.Environment
	for i := range 501 {
		environment = h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: caller.WorkspaceID,
			ProjectID:   caller.ProjectID,
			AppID:       target.AppID,
			Slug:        fmt.Sprintf("candidate-%03d", i),
		})
		environments = append(environments, openapi.ConnectionTargetEnvironment{
			Id:    environment.ID,
			AppId: environment.AppID,
			Slug:  environment.Slug,
			Kind:  openapi.ConnectionTargetEnvironmentKind(environment.Kind),
		})
	}
	request := handler.Request{Project: caller.ProjectID, App: caller.AppID, Environment: caller.ID}
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))
	all := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, all.Status, "%s", all.RawBody)
	require.Equal(t, apps[:100], all.Body.Data.Apps)
	require.Equal(t, environments[:500], all.Body.Data.Environments)
	require.NotNil(t, all.Body.Data.Deployments)
	require.Empty(t, all.Body.Data.Deployments)

	headers = authHeaders(h.CreateRootKey(caller.WorkspaceID, environmentScope(caller)+"#read", appScope(target)+"#read", environmentScope(environment)+"#read"))
	filtered := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, filtered.Status, "%s", filtered.RawBody)
	require.NotNil(t, filtered.Body.Data.Apps)
	require.Empty(t, filtered.Body.Data.Apps)
	require.NotNil(t, filtered.Body.Data.Environments)
	require.Empty(t, filtered.Body.Data.Environments)
	require.False(t, filtered.Body.Pagination.HasMore)
	require.Nil(t, filtered.Body.Pagination.Cursor)
}

func TestListTargetsDoesNotIncludePinsFromOtherCallersOrResourcesOutsideProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	caller := seedCaller(t, h, h.Resources().UserWorkspace.ID)
	target := seedTarget(t, h, caller, "target")
	otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: caller.WorkspaceID,
		ProjectID:   caller.ProjectID,
		AppID:       caller.AppID,
		Slug:        "other",
	})
	deployment := seedDeployment(t, h, target, mysqltype.DeploymentsStatusReady, 100, true)
	seedPin(t, h, otherEnvironment, deployment, "other-environment-pin")
	seedPin(t, h, target, seedDeployment(t, h, caller, mysqltype.DeploymentsStatusReady, 100, true), "other-app-pin")
	otherProject := seedCaller(t, h, caller.WorkspaceID)
	foreign := seedCaller(t, h, h.CreateWorkspace().ID)
	for _, environment := range []db.Environment{otherProject, foreign} {
		targetOther := seedTarget(t, h, environment, "other-target")
		seedPin(t, h, environment, seedDeployment(t, h, targetOther, mysqltype.DeploymentsStatusReady, 100, true), "outside-project-pin")
	}
	headers := authHeaders(h.CreateRootKey(caller.WorkspaceID, "unkey:v1:"+caller.WorkspaceID+":**#*"))
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project:     caller.ProjectID,
		App:         caller.AppID,
		Environment: caller.ID,
	})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Equal(t, []openapi.ConnectionTargetApp{{Id: target.AppID, Name: "target", Slug: "target"}}, response.Body.Data.Apps)
	require.Len(t, response.Body.Data.Environments, 3)
	var ids []string
	for _, environment := range response.Body.Data.Environments {
		ids = append(ids, environment.Id)
	}
	require.ElementsMatch(t, []string{caller.ID, target.ID, otherEnvironment.ID}, ids)
	require.NotNil(t, response.Body.Data.Deployments)
	require.Empty(t, response.Body.Data.Deployments)
	require.False(t, response.Body.Pagination.HasMore)
	require.Nil(t, response.Body.Pagination.Cursor)
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

func seedDeployment(t *testing.T, h *testutil.Harness, environment db.Environment, status mysqltype.DeploymentsStatus, createdAt int64, ready bool) db.Deployment {
	t.Helper()
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   environment.WorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		Status:        status,
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
