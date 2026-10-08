package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

type appConnectionsFixture struct {
	workspace, otherWorkspace, project, otherProject, foreignProject string
	caller, target, otherTarget, foreign, foreignTarget              string
	environments, deployments, connections                           map[string]string
}

func seedAppConnections(t *testing.T, q *Queries) appConnectionsFixture {
	t.Helper()
	f := appConnectionsFixture{
		workspace: uid.New(uid.WorkspacePrefix), otherWorkspace: uid.New(uid.WorkspacePrefix),
		project: uid.New(uid.ProjectPrefix), otherProject: uid.New(uid.ProjectPrefix), foreignProject: uid.New(uid.ProjectPrefix),
		caller: uid.New(uid.AppPrefix), target: uid.New(uid.AppPrefix), otherTarget: uid.New(uid.AppPrefix),
		foreign: uid.New(uid.AppPrefix), foreignTarget: uid.New(uid.AppPrefix),
		environments: map[string]string{}, deployments: map[string]string{}, connections: map[string]string{},
	}
	insertTestWorkspace(t, q, f.workspace, "namespace")
	insertTestWorkspace(t, q, f.otherWorkspace, "other-namespace")
	insertTestProject(t, q, f.workspace, f.project)
	insertTestProject(t, q, f.workspace, f.otherProject)
	insertTestProject(t, q, f.otherWorkspace, f.foreignProject)
	insertTestApp(t, q, f.workspace, f.project, f.caller, "caller")
	insertTestApp(t, q, f.workspace, f.project, f.target, "target")
	insertTestApp(t, q, f.workspace, f.project, f.otherTarget, "other-target")
	insertTestApp(t, q, f.otherWorkspace, f.foreignProject, f.foreign, "caller")
	insertTestApp(t, q, f.otherWorkspace, f.foreignProject, f.foreignTarget, "target")

	production, preview := mysqltype.EnvironmentKindProduction, mysqltype.EnvironmentKindPreview
	for _, environment := range []struct {
		app, id, slug string
		kind          mysqltype.EnvironmentKind
	}{
		{f.caller, "caller-prod", "production", production},
		{f.caller, "caller-canary", "canary", production},
		{f.caller, "caller-staging", "staging", production},
		{f.caller, "caller-preview", "preview", preview},
		{f.caller, "caller-manual", "manual", preview},
		{f.caller, "caller-pin", "pin", preview},
		{f.caller, "caller-rollback", "rollback", preview},
		{f.target, "target-prod", "production", production},
		{f.target, "target-canary", "canary", production},
		{f.target, "target-preview", "preview", preview},
		{f.target, "target-manual", "manual", preview},
	} {
		id := uid.New(uid.EnvironmentPrefix)
		f.environments[environment.id] = id
		insertTestEnvironment(t, q, f.workspace, f.project, environment.app, id, environment.slug, environment.kind)
	}
	f.environments["foreign-env"] = uid.New(uid.EnvironmentPrefix)
	insertTestEnvironment(t, q, f.otherWorkspace, f.foreignProject, f.foreign, f.environments["foreign-env"], "preview", preview)

	callerDeploymentByEnvironment := map[string]string{}
	for _, environment := range []string{"caller-prod", "caller-canary", "caller-staging", "caller-preview", "caller-manual", "caller-pin", "caller-rollback"} {
		deployment := uid.New(uid.DeploymentPrefix)
		f.deployments[environment] = deployment
		callerDeploymentByEnvironment[f.environments[environment]] = deployment
		insertTestDeployment(t, q, testDeployment{
			ID: deployment, WorkspaceID: f.workspace, ProjectID: f.project, AppID: f.caller, EnvironmentID: f.environments[environment],
			Status: mysqltype.DeploymentsStatusReady, PrivateNetworking: true, CreatedAt: 1,
		})
	}
	f.deployments["target-pinned-stopped"] = uid.New(uid.DeploymentPrefix)
	insertTestDeployment(t, q, testDeployment{
		ID: f.deployments["target-pinned-stopped"], WorkspaceID: f.workspace, ProjectID: f.project, AppID: f.target, EnvironmentID: f.environments["target-preview"],
		Status: mysqltype.DeploymentsStatusStopped, DesiredState: mysqltype.DeploymentsDesiredStateStopped,
		PrivateNetworking: true, FirstReadyAt: sql.NullInt64{Int64: 1, Valid: true}, CreatedAt: 1,
	})

	connection := func(id, name, environment, target, mode, targetEnvironment string) testConnection {
		connectionID := uid.New(uid.ConnectionPrefix)
		f.connections[id] = connectionID
		return testConnection{
			ID: connectionID, WorkspaceID: f.workspace, ProjectID: f.project, AppID: f.caller, EnvironmentID: f.environments[environment],
			ResourceID: target, Name: name, Mode: mode, TargetEnvironmentID: f.environments[targetEnvironment],
		}
	}
	leak := connection("other-project", "leak", "caller-prod", f.otherTarget, "automatic", "")
	leak.ProjectID = f.otherProject
	pinned := connection("pinned", "target", "caller-pin", f.target, "deployment", "")
	pinned.TargetDeploymentID = f.deployments["target-pinned-stopped"]
	for _, c := range []testConnection{
		connection("production", "target", "caller-prod", f.target, "automatic", ""),
		connection("self-prod", "caller", "caller-prod", f.caller, "automatic", ""),
		connection("cross-workspace", "foreign", "caller-prod", f.foreign, "automatic", ""),
		leak,
		connection("canary", "target", "caller-canary", f.target, "environment", "target-canary"),
		connection("canary-auto", "other-target", "caller-canary", f.otherTarget, "automatic", ""),
		connection("self-canary-to-prod", "caller", "caller-canary", f.caller, "environment", "caller-prod"),
		connection("preview", "target", "caller-preview", f.target, "automatic", ""),
		connection("self-preview", "caller", "caller-preview", f.caller, "automatic", ""),
		connection("manual", "target", "caller-manual", f.target, "environment", "target-manual"),
		connection("reserved-prefix", "unkey-internal", "caller-manual", f.otherTarget, "automatic", ""),
		pinned,
		connection("reserved-own-slug", "caller", "caller-pin", f.otherTarget, "automatic", ""),
		connection("explicit-prod", "target", "caller-rollback", f.target, "environment", "target-prod"),
		connection("staging-auto", "target", "caller-staging", f.target, "automatic", ""),
	} {
		insertTestDefaultConnection(t, q, c)
		require.Contains(t, callerDeploymentByEnvironment, c.EnvironmentID)
		insertTestSavedConnection(t, q, callerDeploymentByEnvironment[c.EnvironmentID], c)
	}

	f.connections["foreign-connection"] = uid.New(uid.ConnectionPrefix)
	foreign := testConnection{
		ID: f.connections["foreign-connection"], WorkspaceID: f.otherWorkspace, ProjectID: f.foreignProject, AppID: f.foreign,
		EnvironmentID: f.environments["foreign-env"], ResourceID: f.foreignTarget, Name: "target", Mode: "automatic",
	}
	insertTestDefaultConnection(t, q, foreign)
	for range 2 {
		deployment := uid.New(uid.DeploymentPrefix)
		insertTestDeployment(t, q, testDeployment{
			ID: deployment, WorkspaceID: f.otherWorkspace, ProjectID: f.foreignProject, AppID: f.foreign, EnvironmentID: f.environments["foreign-env"],
			Status: mysqltype.DeploymentsStatusReady, PrivateNetworking: true, CreatedAt: 1,
		})
		insertTestSavedConnection(t, q, deployment, foreign)
	}
	return f
}

func appConnectionIDs(t *testing.T, q *Queries, workspace, project, app, environment string) []string {
	t.Helper()
	connections, err := q.ListAppConnectionsByApp(t.Context(), ListAppConnectionsByAppParams{
		WorkspaceID: workspace, ProjectID: project, AppID: app, EnvironmentID: environment,
	})
	require.NoError(t, err)
	ids := make([]string, 0, len(connections))
	for _, connection := range connections {
		ids = append(ids, connection.ID)
	}
	return ids
}

// TestAppConnections guarantees that saved app connections keep a single
// target, that deleting an owner never leaves target settings behind, that
// running deployments keep the targets they saved, and that secret injection
// sees only the caller environment's app connections.
func TestAppConnections(t *testing.T) {
	database := openTestDatabase(t)
	newFixture := func(t *testing.T) (appConnectionsFixture, *Queries) {
		t.Helper()
		tx := beginRollbackTx(t, database)
		q := NewQueries(tx)
		return seedAppConnections(t, q), q
	}
	insertNonAppConnections := func(t *testing.T, q *Queries, f appConnectionsFixture) {
		t.Helper()
		for _, resourceType := range []string{"queue", "vault"} {
			insertTestDefaultConnection(t, q, testConnection{
				ID: uid.New(uid.ConnectionPrefix), WorkspaceID: f.workspace, ProjectID: f.project, AppID: f.caller, EnvironmentID: f.environments["caller-canary"],
				ResourceType: resourceType, ResourceID: f.target, Name: resourceType,
			})
		}
	}

	t.Run("app targets are one to one with their saved connection", func(t *testing.T) {
		f, q := newFixture(t)
		err := q.InsertConnectionAppTarget(t.Context(), InsertConnectionAppTargetParams{
			ConnectionID: f.connections["production"], SelectionMode: ConnectionAppTargetsSelectionModeAutomatic,
		})
		require.True(t, IsDuplicateKeyError(err), "a default must not have multiple app targets: %v", err)
		err = q.InsertDeploymentConnectionAppTarget(t.Context(), InsertDeploymentConnectionAppTargetParams{
			DeploymentID: f.deployments["caller-prod"], ConnectionID: f.connections["production"], SelectionMode: DeploymentConnectionAppTargetsSelectionModeAutomatic,
		})
		require.True(t, IsDuplicateKeyError(err), "a deployment snapshot must not have multiple app targets: %v", err)
	})

	for _, scope := range []string{"app", "environment", "project", "workspace"} {
		t.Run(scope+" deletion leaves no orphan app targets", func(t *testing.T) {
			f, q := newFixture(t)
			switch scope {
			case "app":
				require.NoError(t, q.DeleteAppById(t.Context(), f.caller))
			case "environment":
				require.NoError(t, q.DeleteDeploymentConnectionsByEnvironmentId(t.Context(), DeleteDeploymentConnectionsByEnvironmentIdParams{AppID: f.caller, EnvironmentID: f.environments["caller-prod"]}))
				require.NoError(t, q.DeleteEnvironmentById(t.Context(), f.environments["caller-prod"]))
				count, err := q.CountDeploymentConnectionsByEnvironmentId(t.Context(), f.environments["caller-prod"])
				require.NoError(t, err)
				require.Zero(t, count,
					"the deleted environment's saved deployment connections must be gone")
			case "project":
				require.NoError(t, q.DeleteProjectById(t.Context(), f.project))
			case "workspace":
				require.NoError(t, q.DeleteDeploymentConnectionsByWorkspaceIds(t.Context(), []string{f.workspace}))
				require.NoError(t, q.DeleteWorkspacesWithChildren(t.Context(), []string{f.workspace}))
				count, err := q.CountDeploymentConnectionsByWorkspaceId(t.Context(), f.workspace)
				require.NoError(t, err)
				require.Zero(t, count,
					"the deleted workspace's saved deployment connections must be gone")
			}
			orphans, err := q.CountOrphanConnectionAppTargets(t.Context())
			require.NoError(t, err)
			require.Zero(t, orphans)
			orphans, err = q.CountOrphanDeploymentConnectionAppTargets(t.Context())
			require.NoError(t, err)
			require.Zero(t, orphans)

			foreign, err := q.ListAppConnectionsByApp(t.Context(), ListAppConnectionsByAppParams{
				WorkspaceID: f.otherWorkspace, ProjectID: f.foreignProject, AppID: f.foreign, EnvironmentID: f.environments["foreign-env"],
			})
			require.NoError(t, err)
			require.Len(t, foreign, 1)
			require.Equal(t, NullConnectionAppTargetsSelectionMode{
				ConnectionAppTargetsSelectionMode: ConnectionAppTargetsSelectionModeAutomatic, Valid: true,
			}, foreign[0].SelectionMode, "another workspace's target settings must survive")
			saved, err := q.CountDeploymentConnectionAppTargetsByConnectionId(t.Context(), f.connections["foreign-connection"])
			require.NoError(t, err)
			require.Equal(t, int64(2), saved,
				"another workspace's saved targets must survive")
		})
	}

	t.Run("secret injection lists the caller environment's app connections", func(t *testing.T) {
		f, q := newFixture(t)
		for environment, expectedIDs := range map[string][]string{
			"caller-prod":    {f.connections["production"], f.connections["cross-workspace"]},
			"caller-preview": {f.connections["preview"]},
			"caller-canary":  {f.connections["canary"], f.connections["canary-auto"]},
			"caller-staging": {f.connections["staging-auto"]},
			"caller-pin":     {f.connections["pinned"]},
			"target-prod":    {},
		} {
			require.ElementsMatch(t, expectedIDs, appConnectionIDs(t, q, f.workspace, f.project, f.caller, f.environments[environment]), "secret injection for %s", environment)
		}
	})

	t.Run("only app connections inject private DNS hostnames", func(t *testing.T) {
		f, q := newFixture(t)
		insertNonAppConnections(t, q, f)
		require.ElementsMatch(t, []string{f.connections["canary"], f.connections["canary-auto"]}, appConnectionIDs(t, q, f.workspace, f.project, f.caller, f.environments["caller-canary"]))
	})

	t.Run("deleting a target environment keeps the connection", func(t *testing.T) {
		f, q := newFixture(t)
		require.NoError(t, q.DeleteEnvironmentById(t.Context(), f.environments["target-canary"]))
		count, err := q.CountAppConnectionsByWorkspaceId(t.Context(), f.workspace)
		require.NoError(t, err)
		require.Equal(t, int64(15), count)
		require.ElementsMatch(t, []string{f.connections["canary"], f.connections["canary-auto"]}, appConnectionIDs(t, q, f.workspace, f.project, f.caller, f.environments["caller-canary"]))
	})

	t.Run("deleting a caller environment removes its connections", func(t *testing.T) {
		f, q := newFixture(t)
		require.NoError(t, q.DeleteEnvironmentById(t.Context(), f.environments["caller-prod"]))
		count, err := q.CountAppConnectionsByWorkspaceId(t.Context(), f.workspace)
		require.NoError(t, err)
		require.Equal(t, int64(11), count)
	})

	t.Run("deleting an app keeps other resource types with the same ID", func(t *testing.T) {
		f, q := newFixture(t)
		insertNonAppConnections(t, q, f)
		require.NoError(t, q.DeleteAppById(t.Context(), f.target))
		count, err := q.CountAppConnectionsByResource(t.Context(), CountAppConnectionsByResourceParams{ResourceID: f.target, ResourceType: "app"})
		require.NoError(t, err)
		require.Zero(t, count)
		count, err = q.CountAppConnectionsByResource(t.Context(), CountAppConnectionsByResourceParams{ResourceID: f.target})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
	})

	t.Run("deleting the caller removes connections of every resource type", func(t *testing.T) {
		f, q := newFixture(t)
		insertNonAppConnections(t, q, f)
		require.NoError(t, q.DeleteAppById(t.Context(), f.caller))
		count, err := q.CountAppConnectionsByWorkspaceId(t.Context(), f.workspace)
		require.NoError(t, err)
		require.Zero(t, count)
	})
}
