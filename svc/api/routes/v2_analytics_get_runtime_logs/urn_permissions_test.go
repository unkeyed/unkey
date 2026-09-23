package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test200_CanonicalGlobalGrantReadsWorkspaceLogs(t *testing.T) {
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

func Test200_CanonicalRuntimeLogWildcardReadsWorkspaceLogs(t *testing.T) {
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

// Test200_CanonicalDeploymentGrantUnionPreservesAncestry guarantees independent
// column value lists cannot turn two allowed deployment paths into a Cartesian
// product that exposes crossed ancestry rows.
func Test200_CanonicalDeploymentGrantUnionPreservesAncestry(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	grant := func(projectID, appID, environmentID, deploymentID string) string {
		return fmt.Sprintf(
			"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s/logs#read",
			workspaceID,
			projectID,
			appID,
			environmentID,
			deploymentID,
		)
	}
	rootKey := h.CreateRootKey(workspaceID,
		grant("proj_a", "app_a", "env_a", "dep_a"),
		grant("proj_b", "app_b", "env_b", "dep_b"),
	)

	insertLog(t, h, runtimeLog{logID: "log_allowed_a", workspaceID: workspaceID, projectID: "proj_a", appID: "app_a", environmentID: "env_a", deploymentID: "dep_a"})
	insertLog(t, h, runtimeLog{logID: "log_allowed_b", workspaceID: workspaceID, projectID: "proj_b", appID: "app_b", environmentID: "env_b", deploymentID: "dep_b"})
	insertLog(t, h, runtimeLog{logID: "log_crossed_a", workspaceID: workspaceID, projectID: "proj_a", appID: "app_a", environmentID: "env_a", deploymentID: "dep_b"})
	insertLog(t, h, runtimeLog{logID: "log_crossed_b", workspaceID: workspaceID, projectID: "proj_b", appID: "app_b", environmentID: "env_b", deploymentID: "dep_a"})
	insertLog(t, h, runtimeLog{logID: "log_forbidden", workspaceID: workspaceID, projectID: "proj_c", appID: "app_c", environmentID: "env_c", deploymentID: "dep_c"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT log_id FROM runtime_logs_v1 WHERE log_id = 'not_present' OR 1=1 ORDER BY log_id",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{
		{"log_id": "log_allowed_a"},
		{"log_id": "log_allowed_b"},
	}, res.Body.Data)
}

func Test200_CanonicalAncestorGrantScopesProjectLogs(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/proj_allowed/**#read",
		workspaceID,
	))

	insertLog(t, h, runtimeLog{logID: "log_allowed", workspaceID: workspaceID, projectID: "proj_allowed"})
	insertLog(t, h, runtimeLog{logID: "log_forbidden", workspaceID: workspaceID, projectID: "proj_forbidden"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT log_id FROM runtime_logs_v1 ORDER BY log_id",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"log_id": "log_allowed"}}, res.Body.Data)
}

func Test200_CanonicalGrantWithoutMatchingRowsReturnsEmpty(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/proj_missing/apps/app_missing/environments/env_missing/deployments/dep_missing/logs#read",
		workspaceID,
	))
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "must stay hidden"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT log_id FROM runtime_logs_v1",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
}

func Test200_CanonicalLogDescendantsIncludeTheLogResource(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/proj_a/apps/app_a/environments/env_a/deployments/dep_a/logs/**#read", workspaceID,
	))
	insertLog(t, h, runtimeLog{logID: "allowed", workspaceID: workspaceID, projectID: "proj_a", appID: "app_a", environmentID: "env_a", deploymentID: "dep_a"})
	insertLog(t, h, runtimeLog{logID: "denied", workspaceID: workspaceID, projectID: "proj_a", appID: "app_a", environmentID: "env_a", deploymentID: "dep_b"})
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: "SELECT log_id FROM runtime_logs_v1"})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"log_id": "allowed"}}, res.Body.Data)
}

func Test403_RejectsCanonicalGrantsOutsideRuntimeLogReadScope(t *testing.T) {
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
