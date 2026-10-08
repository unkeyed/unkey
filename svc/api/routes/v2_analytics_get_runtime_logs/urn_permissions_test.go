package handler

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test200_URNGlobalPermissionReadsWorkspaceLogs verifies that **#* reads logs
// from the authorized workspace but not from another workspace.
func Test200_URNGlobalPermissionReadsWorkspaceLogs(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:**#*", workspaceID))

	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "mine"})
	insertLog(t, h, runtimeLog{workspaceID: h.CreateWorkspace().ID, message: "theirs"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT message FROM runtime_logs_v1 ORDER BY message",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"message": "mine"}}, res.Body.Data)
}

// Test200_URNRuntimeLogWildcardReadsWorkspaceLogs verifies that a wildcard
// runtime-log permission reads all logs in its workspace.
func Test200_URNRuntimeLogWildcardReadsWorkspaceLogs(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/*/apps/*/environments/*/deployments/*/logs#read",
		workspaceID,
	))

	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "first"})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "second"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT message FROM runtime_logs_v1 ORDER BY message",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"message": "first"}, {"message": "second"}}, res.Body.Data)
}

// Test200_URNDeploymentPermissionUnionPreservesAncestry guarantees independent
// column value lists cannot turn two allowed deployment paths into a Cartesian
// product that exposes crossed ancestry rows.
func Test200_URNDeploymentPermissionUnionPreservesAncestry(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	permissionFor := func(projectID, appID, environmentID, deploymentID string) string {
		return fmt.Sprintf(
			"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s/logs#read",
			workspaceID,
			projectID,
			appID,
			environmentID,
			deploymentID,
		)
	}
	projectA, appA, envA, deploymentA := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix), uid.New(uid.DeploymentPrefix)
	projectB, appB, envB, deploymentB := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix), uid.New(uid.DeploymentPrefix)
	rootKey := h.CreateRootKey(workspaceID,
		permissionFor(projectA, appA, envA, deploymentA),
		permissionFor(projectB, appB, envB, deploymentB),
	)

	allowedA := insertLog(t, h, runtimeLog{workspaceID: workspaceID, projectID: projectA, appID: appA, environmentID: envA, deploymentID: deploymentA})
	allowedB := insertLog(t, h, runtimeLog{workspaceID: workspaceID, projectID: projectB, appID: appB, environmentID: envB, deploymentID: deploymentB})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, projectID: projectA, appID: appA, environmentID: envA, deploymentID: deploymentB})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, projectID: projectB, appID: appB, environmentID: envB, deploymentID: deploymentA})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT log_id FROM runtime_logs_v1 WHERE log_id = 'not_present' OR 1=1 ORDER BY log_id",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	allowedLogIDs := []string{allowedA.logID, allowedB.logID}
	slices.Sort(allowedLogIDs)
	require.Equal(t, []map[string]any{
		{"log_id": allowedLogIDs[0]},
		{"log_id": allowedLogIDs[1]},
	}, res.Body.Data)
}

// Test200_URNAncestorPermissionScopesProjectLogs verifies that a project
// descendant permission reads that project's logs but not a sibling's logs.
func Test200_URNAncestorPermissionScopesProjectLogs(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	allowedProjectID := uid.New(uid.ProjectPrefix)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/**#read",
		workspaceID,
		allowedProjectID,
	))

	allowed := insertLog(t, h, runtimeLog{workspaceID: workspaceID, projectID: allowedProjectID})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT log_id FROM runtime_logs_v1 ORDER BY log_id",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"log_id": allowed.logID}}, res.Body.Data)
}

// Test200_URNPermissionWithoutMatchingRowsReturnsEmpty verifies that a valid
// permission for a missing deployment returns no logs instead of all logs.
func Test200_URNPermissionWithoutMatchingRowsReturnsEmpty(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s/logs#read",
		workspaceID,
		uid.New(uid.ProjectPrefix),
		uid.New(uid.AppPrefix),
		uid.New(uid.EnvironmentPrefix),
		uid.New(uid.DeploymentPrefix),
	))
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "must stay hidden"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT log_id FROM runtime_logs_v1",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
}

// Test200_URNLogDescendantsIncludeTheLogResource verifies that logs/** includes
// the logs resource itself while preserving deployment scope.
func Test200_URNLogDescendantsIncludeTheLogResource(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	projectID, appID, environmentID, deploymentID := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix), uid.New(uid.DeploymentPrefix)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s/logs/**#read", workspaceID, projectID, appID, environmentID, deploymentID,
	))
	allowed := insertLog(t, h, runtimeLog{workspaceID: workspaceID, projectID: projectID, appID: appID, environmentID: environmentID, deploymentID: deploymentID})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, projectID: projectID, appID: appID, environmentID: environmentID})
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: "SELECT log_id FROM runtime_logs_v1"})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"log_id": allowed.logID}}, res.Body.Data)
}

// Test403_RejectsURNPermissionsOutsideRuntimeLogReadScope verifies that empty,
// wrong-action, wrong-workspace, and wrong-resource permissions are rejected.
func Test403_RejectsURNPermissionsOutsideRuntimeLogReadScope(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	otherWorkspaceID := h.CreateWorkspace().ID

	for name, permissions := range map[string][]string{
		"empty resolution": {},
		"wrong action": {
			fmt.Sprintf("unkey:v1:%s:projects/*/apps/*/environments/*/deployments/*/logs#write", workspaceID),
		},
		"wrong workspace": {
			fmt.Sprintf("unkey:v1:%s:projects/*/apps/*/environments/*/deployments/*/logs#read", otherWorkspaceID),
		},
		"wrong resource": {
			fmt.Sprintf("unkey:v1:%s:projects/*/keyspaces/*/logs#read", workspaceID),
		},
	} {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, permissions...)
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
				Query: "SELECT count() FROM runtime_logs_v1",
			})
			require.Equal(t, 403, res.Status, "response: %s", res.RawBody)
		})
	}
}
