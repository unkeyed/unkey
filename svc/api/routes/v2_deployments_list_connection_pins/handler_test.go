package handler_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_connection_pins"
)

func TestListConnectionPinsReturnsCurrentRulesWithoutSnapshotFields(t *testing.T) {
	h, route, target, caller := setup(t)
	id := seedCurrent(t, h, caller, target, "saved-rule", db.InsertConnectionAppTargetParams{})
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, deploymentScope(target)+"#read"))
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{DeploymentId: target.ID})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.NotEmpty(t, response.Body.Meta.RequestId)
	require.False(t, response.Body.Data.Truncated)
	require.Len(t, response.Body.Data.Pins, 1)
	pin, err := response.Body.Data.Pins[0].AsCurrentConnectionPin()
	require.NoError(t, err)
	require.Equal(t, openapi.CurrentConnectionPin{
		ConnectionId:   id,
		ConnectionName: "saved-rule",
		CallerAppId:    caller.AppID,
		CallerAppName:  "Caller",
		Provenance:     openapi.CurrentRule,
	}, pin)
	var wire struct {
		Data struct {
			Pins []map[string]json.RawMessage `json:"pins"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(response.RawBody), &wire))
	require.Len(t, wire.Data.Pins, 1)
	require.NotContains(t, wire.Data.Pins[0], "callerDeploymentId")
	require.NotContains(t, wire.Data.Pins[0], "callerStatus")
}

func TestListConnectionPinsSortsCurrentRulesAndTruncatesOnlyAboveTen(t *testing.T) {
	h, route, target, caller := setup(t)
	ids := make([]string, 10)
	for i := len(ids) - 1; i >= 0; i-- {
		environment := seedEnvironment(t, h, caller, fmt.Sprintf("preview-%02d", i))
		ids[i] = seedCurrent(t, h, environment, target, fmt.Sprintf("rule-%02d", i), db.InsertConnectionAppTargetParams{})
	}
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, deploymentScope(target)+"#read"))
	request := handler.Request{DeploymentId: target.ID}
	ten := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, ten.Status, "%s", ten.RawBody)
	require.False(t, ten.Body.Data.Truncated)
	require.Len(t, ten.Body.Data.Pins, 10)
	for i, value := range ten.Body.Data.Pins {
		pin, err := value.AsCurrentConnectionPin()
		require.NoError(t, err)
		require.Equal(t, ids[i], pin.ConnectionId)
		require.Equal(t, fmt.Sprintf("rule-%02d", i), pin.ConnectionName)
		require.Equal(t, openapi.CurrentRule, pin.Provenance)
	}
	seedCurrent(t, h, caller, target, "rule-10", db.InsertConnectionAppTargetParams{})
	eleven := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, eleven.Status, "%s", eleven.RawBody)
	require.True(t, eleven.Body.Data.Truncated)
	require.Equal(t, ten.Body.Data.Pins, eleven.Body.Data.Pins)
}

func TestListConnectionPinsRequiresMatchingCurrentCallerAndTarget(t *testing.T) {
	h, route, target, caller := setup(t)
	for _, field := range []string{"workspace", "project", "app", "target app", "target deployment", "automatic", "environment"} {
		invalidCaller, invalidTarget := seedEnvironment(t, h, caller, "invalid-"+field), target
		settings := db.InsertConnectionAppTargetParams{}
		switch field {
		case "workspace":
			invalidCaller.WorkspaceID = h.Resources().RootWorkspace.ID
		case "project":
			invalidCaller.ProjectID = uid.New(uid.ProjectPrefix)
		case "app":
			invalidCaller.AppID = uid.New(uid.AppPrefix)
		case "target app":
			invalidTarget.AppID = uid.New(uid.AppPrefix)
		case "target deployment":
			invalidTarget.ID = uid.New(uid.DeploymentPrefix)
		case "automatic":
			settings.SelectionMode = db.ConnectionAppTargetsSelectionModeAutomatic
		case "environment":
			settings.SelectionMode = db.ConnectionAppTargetsSelectionModeEnvironment
			settings.TargetEnvironmentID = sql.NullString{String: target.EnvironmentID, Valid: true}
		}
		seedCurrent(t, h, invalidCaller, invalidTarget, "invalid-"+field, settings)
	}
	for _, field := range []string{"workspace", "project"} {
		app := seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: target.WorkspaceID,
			ProjectID:   target.ProjectID,
			Name:        "Invalid caller",
			Slug:        "invalid-" + field,
		}
		if field == "workspace" {
			app.WorkspaceID = h.Resources().RootWorkspace.ID
		} else {
			app.ProjectID = uid.New(uid.ProjectPrefix)
		}
		invalidApp := h.CreateApp(app)
		invalidCaller := caller
		invalidCaller.AppID = invalidApp.ID
		seedCurrent(t, h, invalidCaller, target, "invalid-app-"+field, db.InsertConnectionAppTargetParams{})
	}
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, deploymentScope(target)+"#read"))
	request := handler.Request{DeploymentId: target.ID}
	empty := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, empty.Status, "%s", empty.RawBody)
	require.NotNil(t, empty.Body.Data.Pins)
	require.Empty(t, empty.Body.Data.Pins)
	require.False(t, empty.Body.Data.Truncated)
	id := seedCurrent(t, h, caller, target, "valid", db.InsertConnectionAppTargetParams{})
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Len(t, response.Body.Data.Pins, 1)
	pin, err := response.Body.Data.Pins[0].AsCurrentConnectionPin()
	require.NoError(t, err)
	require.Equal(t, id, pin.ConnectionId)
}

func TestListConnectionPinsExcludesSnapshotWithFalsePrivateNetworkingCapability(t *testing.T) {
	h, route, target, caller := setup(t)
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   caller.WorkspaceID,
		ProjectID:     caller.ProjectID,
		AppID:         caller.AppID,
		EnvironmentID: caller.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		DesiredState:  mysqltype.DeploymentsDesiredStateRunning,
	})
	require.False(t, deployment.Capabilities.PrivateNetworking)
	id := uid.New(uid.ConnectionPrefix)
	require.NoError(t, db.Query.InsertDeploymentConnection(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionParams{
		DeploymentID:  deployment.ID,
		ConnectionID:  id,
		WorkspaceID:   caller.WorkspaceID,
		ProjectID:     caller.ProjectID,
		AppID:         caller.AppID,
		EnvironmentID: caller.ID,
		ResourceType:  db.DeploymentConnectionsResourceTypeApp,
		ResourceID:    target.AppID,
		Name:          "saved",
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertDeploymentConnectionAppTarget(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionAppTargetParams{
		DeploymentID:       deployment.ID,
		ConnectionID:       id,
		SelectionMode:      db.DeploymentConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: target.ID, Valid: true},
	}))
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, deploymentScope(target)+"#read"))
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{DeploymentId: target.ID})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.NotNil(t, response.Body.Data.Pins)
	require.Empty(t, response.Body.Data.Pins)
	require.False(t, response.Body.Data.Truncated)
}

func TestListConnectionPinsKeepsCurrentAndSnapshotSeparateAndSnapshotSurvivesDeletion(t *testing.T) {
	h, route, target, caller := setup(t)
	id := seedCurrent(t, h, caller, target, "z-current", db.InsertConnectionAppTargetParams{})
	deployment := seedCaller(t, h, caller, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
	seedSnapshot(t, h, deployment, target, "a-snapshot", db.InsertDeploymentConnectionAppTargetParams{ConnectionID: id})
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, deploymentScope(target)+"#read"))
	request := handler.Request{DeploymentId: target.ID}
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.False(t, response.Body.Data.Truncated)
	require.Len(t, response.Body.Data.Pins, 2)
	current, err := response.Body.Data.Pins[0].AsCurrentConnectionPin()
	require.NoError(t, err)
	require.Equal(t, openapi.CurrentConnectionPin{
		ConnectionId:   id,
		ConnectionName: "z-current",
		CallerAppId:    caller.AppID,
		CallerAppName:  "Caller",
		Provenance:     openapi.CurrentRule,
	}, current)
	snapshot, err := response.Body.Data.Pins[1].AsSnapshotConnectionPin()
	require.NoError(t, err)
	require.Equal(t, openapi.SnapshotConnectionPin{
		ConnectionId:       id,
		ConnectionName:     "a-snapshot",
		CallerAppId:        caller.AppID,
		CallerAppName:      "Caller",
		CallerDeploymentId: deployment.ID,
		CallerStatus:       openapi.DeploymentStatusReady,
		Provenance:         openapi.DeploymentSnapshot,
	}, snapshot)

	require.NoError(t, db.Query.DeleteAppConnection(t.Context(), h.DB.RW(), db.DeleteAppConnectionParams{
		ID:          id,
		WorkspaceID: caller.WorkspaceID,
	}))
	deleted := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, deleted.Status, "%s", deleted.RawBody)
	require.False(t, deleted.Body.Data.Truncated)
	require.Equal(t, response.Body.Data.Pins[1:], deleted.Body.Data.Pins)
}

func TestListConnectionPinsTruncatesCombinedCurrentAndSnapshotResults(t *testing.T) {
	h, route, target, caller := setup(t)
	currentIDs := make([]string, 4)
	for i := len(currentIDs) - 1; i >= 0; i-- {
		environment := seedEnvironment(t, h, caller, fmt.Sprintf("preview-%d", i))
		currentIDs[i] = seedCurrent(t, h, environment, target, fmt.Sprintf("z-current-%d", i), db.InsertConnectionAppTargetParams{})
	}
	snapshotIDs := make([]string, 6)
	for i := len(snapshotIDs) - 1; i >= 0; i-- {
		deployment := seedCaller(t, h, caller, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
		snapshotIDs[i] = seedSnapshot(t, h, deployment, target, fmt.Sprintf("a-snapshot-%d", i), db.InsertDeploymentConnectionAppTargetParams{})
	}
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, deploymentScope(target)+"#read"))
	request := handler.Request{DeploymentId: target.ID}
	ten := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, ten.Status, "%s", ten.RawBody)
	require.Len(t, ten.Body.Data.Pins, 10)
	require.False(t, ten.Body.Data.Truncated)
	for i, id := range currentIDs {
		pin, err := ten.Body.Data.Pins[i].AsCurrentConnectionPin()
		require.NoError(t, err)
		require.Equal(t, id, pin.ConnectionId)
		require.Equal(t, openapi.CurrentRule, pin.Provenance)
	}
	for i, id := range snapshotIDs {
		pin, err := ten.Body.Data.Pins[len(currentIDs)+i].AsSnapshotConnectionPin()
		require.NoError(t, err)
		require.Equal(t, id, pin.ConnectionId)
		require.Equal(t, openapi.DeploymentSnapshot, pin.Provenance)
	}
	deployment := seedCaller(t, h, caller, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
	seedSnapshot(t, h, deployment, target, "a-snapshot-6", db.InsertDeploymentConnectionAppTargetParams{})
	eleven := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, eleven.Status, "%s", eleven.RawBody)
	require.True(t, eleven.Body.Data.Truncated)
	require.Equal(t, ten.Body.Data.Pins, eleven.Body.Data.Pins)
}

func TestListConnectionPinsFiltersSnapshotStatusDesiredStateAndCapability(t *testing.T) {
	h, route, target, caller := setup(t)
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, "unkey:v1:"+target.WorkspaceID+":**#read"))
	for _, test := range []struct {
		status   mysqltype.DeploymentsStatus
		eligible bool
	}{
		{status: mysqltype.DeploymentsStatusPending, eligible: true},
		{status: mysqltype.DeploymentsStatusStarting, eligible: true},
		{status: mysqltype.DeploymentsStatusBuilding, eligible: true},
		{status: mysqltype.DeploymentsStatusDeploying, eligible: true},
		{status: mysqltype.DeploymentsStatusNetwork, eligible: true},
		{status: mysqltype.DeploymentsStatusFinalizing, eligible: true},
		{status: mysqltype.DeploymentsStatusReady, eligible: true},
		{status: mysqltype.DeploymentsStatusFailed},
		{status: mysqltype.DeploymentsStatusSkipped},
		{status: mysqltype.DeploymentsStatusAwaitingApproval},
		{status: mysqltype.DeploymentsStatusStopped},
		{status: mysqltype.DeploymentsStatusSuperseded},
		{status: mysqltype.DeploymentsStatusCancelled},
	} {
		for _, desired := range []mysqltype.DeploymentsDesiredState{mysqltype.DeploymentsDesiredStateRunning, mysqltype.DeploymentsDesiredStateStopped} {
			for _, capability := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/%s/private-networking=%t", test.status, desired, capability), func(t *testing.T) {
					target := h.CreateDeployment(seed.CreateDeploymentRequest{
						ID:            uid.New(uid.DeploymentPrefix),
						WorkspaceID:   target.WorkspaceID,
						ProjectID:     target.ProjectID,
						AppID:         target.AppID,
						EnvironmentID: target.EnvironmentID,
					})
					deployment := seedCaller(t, h, caller, test.status, desired, capability)
					id := seedSnapshot(t, h, deployment, target, "saved", db.InsertDeploymentConnectionAppTargetParams{})
					response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{DeploymentId: target.ID})
					require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
					require.NotNil(t, response.Body.Data.Pins)
					require.False(t, response.Body.Data.Truncated)
					if !test.eligible || desired != mysqltype.DeploymentsDesiredStateRunning || !capability {
						require.Empty(t, response.Body.Data.Pins)
						return
					}
					require.Len(t, response.Body.Data.Pins, 1)
					pin, err := response.Body.Data.Pins[0].AsSnapshotConnectionPin()
					require.NoError(t, err)
					require.Equal(t, openapi.SnapshotConnectionPin{
						ConnectionId:       id,
						ConnectionName:     "saved",
						CallerAppId:        caller.AppID,
						CallerAppName:      "Caller",
						CallerDeploymentId: deployment.ID,
						CallerStatus:       openapi.DeploymentStatus(test.status),
						Provenance:         openapi.DeploymentSnapshot,
					}, pin)
				})
			}
		}
	}
}

func TestListConnectionPinsRequiresCompleteSnapshotCallerIdentityAndCompositeSettings(t *testing.T) {
	h, route, target, caller := setup(t)
	for _, field := range []string{"workspace", "project", "app", "environment", "deployment", "target app", "target deployment", "automatic", "environment mode", "settings deployment", "settings connection"} {
		deployment := seedCaller(t, h, caller, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
		invalidCaller, invalidTarget := deployment, target
		settings := db.InsertDeploymentConnectionAppTargetParams{}
		switch field {
		case "workspace":
			invalidCaller.WorkspaceID = h.Resources().RootWorkspace.ID
		case "project":
			invalidCaller.ProjectID = uid.New(uid.ProjectPrefix)
		case "app":
			invalidCaller.AppID = target.AppID
		case "environment":
			invalidCaller.EnvironmentID = seedEnvironment(t, h, caller, "other").ID
		case "deployment":
			invalidCaller.ID = uid.New(uid.DeploymentPrefix)
		case "target app":
			invalidTarget.AppID = uid.New(uid.AppPrefix)
		case "target deployment":
			invalidTarget.ID = uid.New(uid.DeploymentPrefix)
		case "automatic":
			settings.SelectionMode = db.DeploymentConnectionAppTargetsSelectionModeAutomatic
		case "environment mode":
			settings.SelectionMode = db.DeploymentConnectionAppTargetsSelectionModeEnvironment
			settings.TargetEnvironmentID = sql.NullString{String: target.EnvironmentID, Valid: true}
		case "settings deployment":
			other := seedCaller(t, h, caller, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
			settings.DeploymentID = other.ID
		case "settings connection":
			insertSnapshot(t, h, deployment, target, uid.New(uid.ConnectionPrefix), "missing-settings")
			require.NoError(t, db.Query.InsertDeploymentConnectionAppTarget(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionAppTargetParams{
				DeploymentID:       deployment.ID,
				ConnectionID:       uid.New(uid.ConnectionPrefix),
				SelectionMode:      db.DeploymentConnectionAppTargetsSelectionModeDeployment,
				TargetDeploymentID: sql.NullString{String: target.ID, Valid: true},
			}))
			continue
		}
		seedSnapshot(t, h, invalidCaller, invalidTarget, "invalid-"+field, settings)
	}
	for _, field := range []string{"workspace", "project", "app", "environment"} {
		identity := caller
		switch field {
		case "workspace":
			identity.WorkspaceID = h.Resources().RootWorkspace.ID
		case "project":
			identity.ProjectID = h.CreateProject(seed.CreateProjectRequest{
				ID:          uid.New(uid.ProjectPrefix),
				WorkspaceID: caller.WorkspaceID,
				Name:        "Other project",
				Slug:        "other-project",
			}).ID
		case "app":
			identity.AppID = target.AppID
		case "environment":
			identity.ID = seedEnvironment(t, h, caller, "other-deployment").ID
		}
		deployment := seedCaller(t, h, identity, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
		deployment.WorkspaceID = caller.WorkspaceID
		deployment.ProjectID = caller.ProjectID
		deployment.AppID = caller.AppID
		deployment.EnvironmentID = caller.ID
		seedSnapshot(t, h, deployment, target, "invalid-deployment-"+field, db.InsertDeploymentConnectionAppTargetParams{})
	}
	for _, field := range []string{"environment workspace", "environment app", "app workspace", "app project"} {
		identity := caller
		if field == "environment workspace" || field == "environment app" {
			environment := caller
			if field == "environment workspace" {
				environment.WorkspaceID = h.Resources().RootWorkspace.ID
			} else {
				environment.AppID = target.AppID
			}
			identity.ID = seedEnvironment(t, h, environment, "invalid-"+field).ID
		} else {
			app := seed.CreateAppRequest{
				ID:          uid.New(uid.AppPrefix),
				WorkspaceID: caller.WorkspaceID,
				ProjectID:   caller.ProjectID,
				Name:        "Invalid caller",
				Slug:        "invalid-" + field,
			}
			if field == "app workspace" {
				app.WorkspaceID = h.Resources().RootWorkspace.ID
			} else {
				app.ProjectID = uid.New(uid.ProjectPrefix)
			}
			identity.AppID = h.CreateApp(app).ID
			identity = seedEnvironment(t, h, identity, "preview")
		}
		deployment := seedCaller(t, h, identity, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
		seedSnapshot(t, h, deployment, target, "invalid-"+field, db.InsertDeploymentConnectionAppTargetParams{})
	}
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, deploymentScope(target)+"#read"))
	request := handler.Request{DeploymentId: target.ID}
	empty := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, empty.Status, "%s", empty.RawBody)
	require.NotNil(t, empty.Body.Data.Pins)
	require.Empty(t, empty.Body.Data.Pins)
	require.False(t, empty.Body.Data.Truncated)
	deployment := seedCaller(t, h, caller, mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsDesiredStateRunning, true)
	id := seedSnapshot(t, h, deployment, target, "valid", db.InsertDeploymentConnectionAppTargetParams{})
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Len(t, response.Body.Data.Pins, 1)
	pin, err := response.Body.Data.Pins[0].AsSnapshotConnectionPin()
	require.NoError(t, err)
	require.Equal(t, id, pin.ConnectionId)
	require.Equal(t, deployment.ID, pin.CallerDeploymentId)
	require.Equal(t, openapi.DeploymentSnapshot, pin.Provenance)
}

func TestListConnectionPinsRequiresExactTargetDeploymentReadAndMasksNotFound(t *testing.T) {
	h, route, target, caller := setup(t)
	seedCurrent(t, h, caller, target, "caller", db.InsertConnectionAppTargetParams{})
	headers := authHeaders(h.CreateRootKey(target.WorkspaceID, "unkey:v1:"+target.WorkspaceID+":**#*"))
	missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{DeploymentId: uid.New(uid.DeploymentPrefix)})
	require.Equal(t, http.StatusNotFound, missing.Status, "%s", missing.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", missing.Body.Error.Type)
	other := target
	other.ID = uid.New(uid.DeploymentPrefix)
	for _, test := range []struct {
		name        string
		workspaceID string
		permissions []string
	}{
		{name: "no read", workspaceID: target.WorkspaceID},
		{name: "write only", workspaceID: target.WorkspaceID, permissions: []string{deploymentScope(target) + "#write"}},
		{name: "other deployment read", workspaceID: target.WorkspaceID, permissions: []string{deploymentScope(other) + "#read"}},
		{name: "caller app read", workspaceID: target.WorkspaceID, permissions: []string{"unkey:v1:" + caller.WorkspaceID + ":projects/" + caller.ProjectID + "/apps/" + caller.AppID + "/**#read"}},
		{name: "foreign workspace", workspaceID: h.Resources().RootWorkspace.ID, permissions: []string{"unkey:v1:" + h.Resources().RootWorkspace.ID + ":**#*"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(test.workspaceID, test.permissions...))
			response := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{DeploymentId: target.ID})
			require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
			require.Equal(t, missing.Body.Error, response.Body.Error)
		})
	}
}

func setup(t *testing.T) (*testutil.Harness, *handler.Handler, db.Deployment, db.Environment) {
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
	var environments []db.Environment
	for _, name := range []string{"Target", "Caller"} {
		app := h.CreateApp(seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: project.WorkspaceID,
			ProjectID:   project.ID,
			Name:        name,
			Slug:        name,
		})
		environments = append(environments, h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: project.WorkspaceID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        "preview",
		}))
	}
	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   project.WorkspaceID,
		ProjectID:     project.ID,
		AppID:         environments[0].AppID,
		EnvironmentID: environments[0].ID,
		Status:        mysqltype.DeploymentsStatusFailed,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
	})
	return h, route, target, environments[1]
}

func seedCurrent(t *testing.T, h *testutil.Harness, caller db.Environment, target db.Deployment, name string, settings db.InsertConnectionAppTargetParams) string {
	t.Helper()
	id := uid.New(uid.ConnectionPrefix)
	require.NoError(t, db.Query.InsertAppConnection(t.Context(), h.DB.RW(), db.InsertAppConnectionParams{
		ID:            id,
		WorkspaceID:   caller.WorkspaceID,
		ProjectID:     caller.ProjectID,
		AppID:         caller.AppID,
		EnvironmentID: caller.ID,
		ResourceType:  db.AppConnectionsResourceTypeApp,
		ResourceID:    target.AppID,
		Name:          name,
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
	settings.ConnectionID = id
	if settings.SelectionMode == "" {
		settings.SelectionMode = db.ConnectionAppTargetsSelectionModeDeployment
	}
	settings.TargetDeploymentID = sql.NullString{String: target.ID, Valid: true}
	require.NoError(t, db.Query.InsertConnectionAppTarget(t.Context(), h.DB.RW(), settings))
	return id
}

func deploymentScope(deployment db.Deployment) string {
	return "unkey:v1:" + deployment.WorkspaceID + ":projects/" + deployment.ProjectID + "/apps/" + deployment.AppID + "/environments/" + deployment.EnvironmentID + "/deployments/" + deployment.ID
}

func seedEnvironment(t *testing.T, h *testutil.Harness, caller db.Environment, slug string) db.Environment {
	t.Helper()
	return h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: caller.WorkspaceID,
		ProjectID:   caller.ProjectID,
		AppID:       caller.AppID,
		Slug:        slug,
	})
}

func seedCaller(t *testing.T, h *testutil.Harness, caller db.Environment, status mysqltype.DeploymentsStatus, desired mysqltype.DeploymentsDesiredState, capability bool) db.Deployment {
	t.Helper()
	return h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   caller.WorkspaceID,
		ProjectID:     caller.ProjectID,
		AppID:         caller.AppID,
		EnvironmentID: caller.ID,
		Status:        status,
		DesiredState:  desired,
		Capabilities:  mysqltype.DeploymentCapabilities{PrivateNetworking: capability},
	})
}

func seedSnapshot(t *testing.T, h *testutil.Harness, caller, target db.Deployment, name string, settings db.InsertDeploymentConnectionAppTargetParams) string {
	t.Helper()
	if settings.ConnectionID == "" {
		settings.ConnectionID = uid.New(uid.ConnectionPrefix)
	}
	insertSnapshot(t, h, caller, target, settings.ConnectionID, name)
	if settings.DeploymentID == "" {
		settings.DeploymentID = caller.ID
	}
	if settings.SelectionMode == "" {
		settings.SelectionMode = db.DeploymentConnectionAppTargetsSelectionModeDeployment
	}
	settings.TargetDeploymentID = sql.NullString{String: target.ID, Valid: true}
	require.NoError(t, db.Query.InsertDeploymentConnectionAppTarget(t.Context(), h.DB.RW(), settings))
	return settings.ConnectionID
}

func insertSnapshot(t *testing.T, h *testutil.Harness, caller, target db.Deployment, id, name string) {
	t.Helper()
	require.NoError(t, db.Query.InsertDeploymentConnection(t.Context(), h.DB.RW(), db.InsertDeploymentConnectionParams{
		DeploymentID:  caller.ID,
		ConnectionID:  id,
		WorkspaceID:   caller.WorkspaceID,
		ProjectID:     caller.ProjectID,
		AppID:         caller.AppID,
		EnvironmentID: caller.EnvironmentID,
		ResourceType:  db.DeploymentConnectionsResourceTypeApp,
		ResourceID:    target.AppID,
		Name:          name,
		CreatedAt:     h.Clock.Now().UnixMilli(),
	}))
}

func authHeaders(key string) http.Header {
	return http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
}
