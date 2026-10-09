package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestListPreviewDeploymentsUsedByAppConnections(t *testing.T) {
	tx := beginRollbackTx(t, openTestDatabase(t))
	q := NewQueries(tx)

	workspace, otherWorkspace := uid.New(uid.WorkspacePrefix), uid.New(uid.WorkspacePrefix)
	project, otherProject := uid.New(uid.ProjectPrefix), uid.New(uid.ProjectPrefix)
	target, caller, otherCaller := uid.New(uid.AppPrefix), uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	targetPreview, targetExplicit := uid.New(uid.EnvironmentPrefix), uid.New(uid.EnvironmentPrefix)
	callerPreview, otherPreview := uid.New(uid.EnvironmentPrefix), uid.New(uid.EnvironmentPrefix)
	insertTestWorkspace(t, q, workspace, "namespace")
	insertTestWorkspace(t, q, otherWorkspace, "other-namespace")
	insertTestProject(t, q, workspace, project)
	insertTestProject(t, q, otherWorkspace, otherProject)
	insertTestApp(t, q, workspace, project, target, "target")
	insertTestApp(t, q, workspace, project, caller, "caller")
	insertTestApp(t, q, otherWorkspace, otherProject, otherCaller, "other-caller")
	insertTestEnvironment(t, q, workspace, project, target, targetPreview, "preview", mysqltype.EnvironmentKindPreview)
	insertTestEnvironment(t, q, workspace, project, target, targetExplicit, "explicit", mysqltype.EnvironmentKindPreview)
	insertTestEnvironment(t, q, workspace, project, caller, callerPreview, "preview", mysqltype.EnvironmentKindPreview)
	insertTestEnvironment(t, q, otherWorkspace, otherProject, otherCaller, otherPreview, "preview", mysqltype.EnvironmentKindPreview)

	insertDeployment := func(id, workspace, project, app, environment, branch, fork string, desired mysqltype.DeploymentsDesiredState) {
		t.Helper()
		insertTestDeployment(t, q, testDeployment{
			ID: id, WorkspaceID: workspace, ProjectID: project, AppID: app, EnvironmentID: environment,
			GitBranch: branch, ForkRepository: fork, Status: mysqltype.DeploymentsStatusReady,
			DesiredState: desired, PrivateNetworking: true, CreatedAt: 1,
		})
	}

	targetDeployments := map[string]string{}
	for _, branch := range []string{"automatic", "wrong-branch", "fork", "environment", "workspace", "stopped", "zero"} {
		fork := ""
		if branch == "fork" {
			fork = "upstream/fork"
		}
		environment := targetPreview
		if branch == "environment" {
			environment = targetExplicit
		}
		id := uid.New(uid.DeploymentPrefix)
		targetDeployments[branch] = id
		insertDeployment(id, workspace, project, target, environment, branch, fork, mysqltype.DeploymentsDesiredStateRunning)
	}

	running, stopped := mysqltype.DeploymentsDesiredStateRunning, mysqltype.DeploymentsDesiredStateStopped
	callerDeployments := map[string]string{}
	for _, name := range []string{"automatic", "wrong-branch", "fork", "environment", "stopped", "workspace"} {
		callerDeployments[name] = uid.New(uid.DeploymentPrefix)
	}
	insertDeployment(callerDeployments["automatic"], workspace, project, caller, callerPreview, "automatic", "", running)
	insertDeployment(callerDeployments["wrong-branch"], workspace, project, caller, callerPreview, "different", "", running)
	insertDeployment(callerDeployments["fork"], workspace, project, caller, callerPreview, "fork", "other/fork", running)
	insertDeployment(callerDeployments["environment"], workspace, project, caller, callerPreview, "unrelated", "", running)
	insertDeployment(callerDeployments["stopped"], workspace, project, caller, callerPreview, "stopped", "", stopped)
	insertDeployment(callerDeployments["workspace"], otherWorkspace, otherProject, otherCaller, otherPreview, "workspace", "", running)

	saveConnection := func(name, caller, workspace, project, app, environment, mode, targetEnvironment string) {
		t.Helper()
		insertTestSavedConnection(t, q, caller, testConnection{
			ID: uid.New(uid.ConnectionPrefix), WorkspaceID: workspace, ProjectID: project, AppID: app, EnvironmentID: environment,
			ResourceID: target, Name: name, Mode: mode, TargetEnvironmentID: targetEnvironment,
		})
	}
	saveConnection("automatic", callerDeployments["automatic"], workspace, project, caller, callerPreview, "automatic", "")
	saveConnection("wrong-branch", callerDeployments["wrong-branch"], workspace, project, caller, callerPreview, "automatic", "")
	saveConnection("fork", callerDeployments["fork"], workspace, project, caller, callerPreview, "automatic", "")
	saveConnection("environment", callerDeployments["environment"], workspace, project, caller, callerPreview, "environment", targetExplicit)
	saveConnection("stopped", callerDeployments["stopped"], workspace, project, caller, callerPreview, "automatic", "")
	saveConnection("workspace", callerDeployments["workspace"], otherWorkspace, otherProject, otherCaller, otherPreview, "automatic", "")

	candidates := []string{
		targetDeployments["automatic"], targetDeployments["wrong-branch"], targetDeployments["fork"], targetDeployments["environment"],
		targetDeployments["workspace"], targetDeployments["stopped"], targetDeployments["zero"],
	}
	used, err := q.ListPreviewDeploymentsUsedByAppConnections(t.Context(), candidates)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{targetDeployments["automatic"], targetDeployments["environment"]}, used)

	for _, caller := range []string{callerDeployments["automatic"], callerDeployments["environment"]} {
		setTestDeploymentStatus(t, q, caller, mysqltype.DeploymentsStatusAwaitingApproval, sql.NullInt64{})
	}
	used, err = q.ListPreviewDeploymentsUsedByAppConnections(t.Context(), candidates)
	require.NoError(t, err)
	require.Empty(t, used)
}
