package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

func TestListPrivateNetworkAppsSelection(t *testing.T) {
	server := containers.MySQL(t)
	database, err := sql.Open("mysql", server.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	tx, err := database.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tx.Rollback()) })

	exec := func(query string, args ...any) {
		t.Helper()
		_, execErr := tx.ExecContext(t.Context(), query, args...)
		require.NoError(t, execErr)
	}
	exec(`INSERT INTO workspaces (id,org_id,name,slug,k8s_namespace,beta_features) VALUES
		('ws','org-binding-test','Workspace','binding-test','namespace','{}'),
		('other-ws','other-org-binding-test','Other','other-binding-test','other-namespace','{}')`)
	exec(`INSERT INTO projects (id,workspace_id,name,slug,created_at) VALUES
		('project','ws','Project','project',1),('other-project','ws','Other','other',1),
		('foreign-project','other-ws','Foreign','foreign',1)`)
	exec(`INSERT INTO apps (id,workspace_id,project_id,name,slug,source_type,current_deployment_id,created_at) VALUES
		('caller','ws','project','Caller','caller','git','caller-prod-live',1),
		('target','ws','project','Target','target','git','target-live',1),
		('other-target','ws','project','Other Target','other-target','git',NULL,1),
		('foreign','other-ws','foreign-project','Foreign','caller','git','foreign-live',1)`)
	exec(`INSERT INTO environments (id,workspace_id,project_id,app_id,slug,kind,created_at) VALUES
		('caller-prod','ws','project','caller','production','production',1),
		('caller-canary','ws','project','caller','canary','production',1),
		('caller-preview','ws','project','caller','preview','preview',1),
		('caller-manual','ws','project','caller','manual','preview',1),
		('caller-pin','ws','project','caller','pin','preview',1),
		('caller-rollback','ws','project','caller','rollback','preview',1),
		('target-prod','ws','project','target','production','production',1),
		('target-canary','ws','project','target','canary','production',1),
		('target-preview','ws','project','target','preview','preview',1),
		('target-manual','ws','project','target','manual','preview',1),
		('foreign-prod','other-ws','foreign-project','foreign','production','production',1)`)
	exec(`INSERT INTO regions (id,name,platform) VALUES
		('r','binding-region','kubernetes'),('r2','binding-region-2','kubernetes'),('r-other','binding-region-other','other')`)

	insertDeployment := func(id, app, environment, source, branch string, fork any, status, desired string, firstReady any, created int64, regions ...string) {
		t.Helper()
		workspace, project := "ws", "project"
		if app == "foreign" {
			workspace, project = "other-ws", "foreign-project"
		}
		exec(`INSERT INTO deployments (id,k8s_name,workspace_id,project_id,environment_id,app_id,source,git_branch,fork_repository_full_name,sentinel_config,cpu_millicores,memory_mib,desired_state,encrypted_environment_variables,status,first_ready_at,created_at)
			VALUES (?,?,?,?,?,?,?,?,?,'{}',100,128,?,'{}',?,?,?)`, id, id, workspace, project, environment, app, source, sql.NullString{String: branch, Valid: branch != ""}, fork, desired, status, firstReady, created)
		for _, region := range regions {
			exec(`INSERT INTO deployment_topology (workspace_id,deployment_id,region_id,desired_status,created_at) VALUES (?,?,?, 'running',1)`, workspace, id, region)
		}
	}
	insertDeployment("caller-prod-deploying", "caller", "caller-prod", "git", "main", nil, "deploying", "running", nil, 10, "r")
	insertDeployment("caller-prod-live", "caller", "caller-prod", "git", "main", nil, "ready", "running", 1, 9, "r", "r2")
	insertDeployment("caller-canary-oci", "caller", "caller-canary", "oci", "", nil, "ready", "running", 1, 11, "r")
	insertDeployment("caller-preview-own", "caller", "caller-preview", "git", "feature", nil, "ready", "running", 1, 12, "r")
	insertDeployment("caller-preview-fork", "caller", "caller-preview", "git", "feature", "fork/repo", "ready", "running", 1, 13, "r")
	insertDeployment("caller-preview-missing", "caller", "caller-preview", "git", "isolated", nil, "ready", "running", 1, 13, "r")
	insertDeployment("caller-unapproved", "caller", "caller-preview", "git", "feature", nil, "awaiting_approval", "running", nil, 14, "r")
	insertDeployment("caller-wrong-platform", "caller", "caller-preview", "git", "feature", nil, "ready", "running", 1, 15, "r-other")
	insertDeployment("caller-manual-dep", "caller", "caller-manual", "git", "feature", nil, "ready", "running", 1, 16, "r")
	insertDeployment("caller-pin-dep", "caller", "caller-pin", "git", "feature", nil, "ready", "running", 1, 17, "r")
	insertDeployment("caller-rollback-dep", "caller", "caller-rollback", "oci", "", nil, "ready", "running", 1, 18, "r")

	insertDeployment("target-live", "target", "target-prod", "git", "main", nil, "ready", "running", 1, 20, "r")
	insertDeployment("target-canary-ready", "target", "target-canary", "oci", "", nil, "ready", "running", 1, 21, "r")
	insertDeployment("target-new-prod", "target", "target-prod", "git", "main", nil, "ready", "running", 1, 99, "r")
	insertDeployment("target-preview-old", "target", "target-preview", "git", "feature", nil, "ready", "running", 1, 30, "r")
	insertDeployment("target-preview-never", "target", "target-preview", "git", "feature", nil, "ready", "running", nil, 31, "r")
	insertDeployment("target-preview-awaiting", "target", "target-preview", "git", "feature", nil, "awaiting_approval", "running", nil, 32, "r")
	insertDeployment("target-preview-fork", "target", "target-preview", "git", "feature", "fork/repo", "ready", "running", 1, 33, "r")
	insertDeployment("target-manual-old", "target", "target-manual", "git", "manual", nil, "ready", "running", 1, 40, "r")
	insertDeployment("target-manual-latest-failed", "target", "target-manual", "git", "manual", nil, "failed", "running", 1, 41, "r")
	insertDeployment("target-pinned-stopped", "target", "target-preview", "git", "pin", nil, "stopped", "stopped", 1, 50, "r")
	insertDeployment("foreign-live", "foreign", "foreign-prod", "git", "main", nil, "ready", "running", 1, 60, "r")
	exec(`UPDATE deployments SET port = 7946 WHERE id = 'target-live'`)
	exec(`UPDATE deployments SET port = 4000 WHERE id IN ('caller-prod-live', 'caller-prod-deploying')`)

	peers, err := NewQueries(tx).ListPrivateNetworkApps(t.Context(), ListPrivateNetworkAppsParams{Platform: "kubernetes", Limit: 100})
	require.NoError(t, err)
	peerDeployments := make([]string, 0, len(peers))
	for _, peer := range peers {
		require.Equal(t, "unkey-replicas", peer.BindingName)
		require.Equal(t, "self-"+peer.CallerDeploymentID, peer.BindingID)
		require.Equal(t, peer.CallerDeploymentID, peer.DeploymentID)
		peerDeployments = append(peerDeployments, peer.DeploymentID)
	}
	require.ElementsMatch(t, []string{
		"caller-prod-deploying", "caller-prod-live", "caller-canary-oci",
		"caller-preview-own", "caller-preview-fork", "caller-preview-missing",
		"caller-manual-dep", "caller-pin-dep", "caller-rollback-dep",
		"target-live", "target-canary-ready", "target-new-prod", "target-preview-old",
		"target-preview-never", "target-preview-fork", "target-manual-old", "foreign-live",
	}, peerDeployments, "every active deployment gets discovery without any app bindings")

	insertBinding := func(id, name, environment, target, targetType string, targetEnvironment, targetDeployment any) {
		t.Helper()
		exec(`INSERT INTO app_bindings (id,workspace_id,project_id,app_id,environment_id,resource_type,resource_id,name,selection_mode,target_environment_id,target_deployment_id,created_at)
			VALUES (?,'ws','project','caller',?,'app',?,?,?,?,?,1)`, id, environment, target, name, targetType, targetEnvironment, targetDeployment)
	}
	insertBinding("production", "target", "caller-prod", "target", "automatic", nil, nil)
	insertBinding("self-prod", "caller", "caller-prod", "caller", "automatic", nil, nil)
	insertBinding("cross-workspace", "foreign", "caller-prod", "foreign", "automatic", nil, nil)
	insertBinding("canary", "target", "caller-canary", "target", "environment", "target-canary", nil)
	insertBinding("canary-auto", "other-target", "caller-canary", "other-target", "automatic", nil, nil)
	insertBinding("self-canary-to-prod", "caller", "caller-canary", "caller", "environment", "caller-prod", nil)
	insertBinding("preview", "target", "caller-preview", "target", "automatic", nil, nil)
	insertBinding("self-preview", "caller", "caller-preview", "caller", "automatic", nil, nil)
	insertBinding("manual", "target", "caller-manual", "target", "environment", "target-manual", nil)
	insertBinding("pinned", "target", "caller-pin", "target", "deployment", nil, "target-pinned-stopped")
	insertBinding("explicit-prod", "target", "caller-rollback", "target", "environment", "target-prod", nil)
	exec(`INSERT INTO app_bindings (id,workspace_id,project_id,app_id,environment_id,resource_type,resource_id,name,selection_mode,created_at)
		VALUES ('other-project','ws','other-project','caller','caller-prod','app','other-target','leak','automatic',1)`)

	list := func() map[string]ListPrivateNetworkAppsRow {
		t.Helper()
		rows, listErr := NewQueries(tx).ListPrivateNetworkApps(t.Context(), ListPrivateNetworkAppsParams{Platform: "kubernetes", Limit: 100})
		require.NoError(t, listErr)
		byBindingCaller := make(map[string]ListPrivateNetworkAppsRow, len(rows))
		for _, row := range rows {
			if row.BindingName == "unkey-replicas" {
				continue
			}
			key := row.BindingID + "/" + row.CallerDeploymentID
			require.NotContains(t, byBindingCaller, key, "one row per binding and caller deployment, even across regions")
			byBindingCaller[key] = row
			require.Equal(t, row.DeploymentID == "", row.Port == 0, "port is known exactly when %s is resolved", key)
			require.Equal(t, "ws", row.WorkspaceID)
			require.Equal(t, "project", row.ProjectID)
		}
		return byBindingCaller
	}
	selected := list()
	require.Len(t, selected, 10)
	for key, want := range map[string]string{
		"production/caller-prod-deploying":  "target-live",
		"production/caller-prod-live":       "target-live",
		"canary/caller-canary-oci":          "target-canary-ready",
		"canary-auto/caller-canary-oci":     "",
		"preview/caller-preview-own":        "target-preview-old",
		"preview/caller-preview-fork":       "target-preview-fork",
		"preview/caller-preview-missing":    "",
		"manual/caller-manual-dep":          "",
		"pinned/caller-pin-dep":             "",
		"explicit-prod/caller-rollback-dep": "target-live",
	} {
		require.Contains(t, selected, key)
		require.Equal(t, want, selected[key].DeploymentID, key)
	}
	require.Equal(t, int32(7946), selected["production/caller-prod-live"].Port, "port comes from the target deployment")
	for _, excluded := range []string{
		"cross-workspace/caller-prod-live", "other-project/caller-prod-live",
		"preview/caller-unapproved", "preview/caller-wrong-platform",
		"self-preview/caller-unapproved", "self-preview/caller-wrong-platform",
	} {
		require.NotContains(t, selected, excluded)
	}

	exec(`UPDATE apps SET current_deployment_id = 'target-new-prod' WHERE id = 'target'`)
	selected = list()
	require.Equal(t, "target-new-prod", selected["production/caller-prod-live"].DeploymentID)
	require.Equal(t, "target-new-prod", selected["explicit-prod/caller-rollback-dep"].DeploymentID)
	exec(`UPDATE apps SET current_deployment_id = 'target-live' WHERE id = 'target'`)
	selected = list()
	require.Equal(t, "target-live", selected["production/caller-prod-live"].DeploymentID, "automatic production follows a rollback")
	require.Equal(t, "target-live", selected["explicit-prod/caller-rollback-dep"].DeploymentID, "an explicit production environment follows a rollback, not the newest ready deployment")

	exec(`UPDATE apps SET current_deployment_id = NULL WHERE id = 'target'`)
	selected = list()
	require.Empty(t, selected["production/caller-prod-live"].DeploymentID)
	require.Empty(t, selected["explicit-prod/caller-rollback-dep"].DeploymentID, "production must not fall back to a non-live build")
	exec(`UPDATE apps SET current_deployment_id = 'target-live' WHERE id = 'target'`)
	exec(`UPDATE deployments SET status = 'failed' WHERE id = 'target-live'`)
	selected = list()
	require.Empty(t, selected["production/caller-prod-live"].DeploymentID)
	require.Empty(t, selected["explicit-prod/caller-rollback-dep"].DeploymentID, "an unavailable live deployment must not select another production build")
	exec(`UPDATE deployments SET status = 'ready' WHERE id = 'target-live'`)

	for environment, expectedIDs := range map[string][]string{
		"caller-prod":    {"production", "cross-workspace"},
		"caller-preview": {"preview"},
		"caller-canary":  {"canary", "canary-auto"},
		"target-prod":    {},
	} {
		bindings, listErr := NewQueries(tx).ListAppBindingsByApp(t.Context(), ListAppBindingsByAppParams{
			WorkspaceID: "ws", ProjectID: "project", AppID: "caller", EnvironmentID: environment,
		})
		require.NoError(t, listErr)
		ids := make([]string, 0, len(bindings))
		for _, binding := range bindings {
			ids = append(ids, binding.ID)
		}
		require.ElementsMatch(t, expectedIDs, ids, "secret injection for %s", environment)
	}

	rows, err := NewQueries(tx).ListPrivateNetworkApps(t.Context(), ListPrivateNetworkAppsParams{Platform: "missing", Limit: 100})
	require.NoError(t, err)
	require.Empty(t, rows)

	var remaining int
	require.NoError(t, NewQueries(tx).DeleteEnvironmentById(t.Context(), "target-canary"))
	require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM app_bindings`).Scan(&remaining))
	require.Equal(t, 12, remaining, "deleting a target environment keeps the binding visible")
	selected = list()
	require.Contains(t, selected, "canary/caller-canary-oci")
	require.Empty(t, selected["canary/caller-canary-oci"].DeploymentID, "a deleted target environment fails closed even while its deployments linger")
	exec(`INSERT INTO environments (id,workspace_id,project_id,app_id,slug,kind,created_at) VALUES ('target-canary-2','ws','project','target','canary','production',2)`)
	insertDeployment("target-canary-recreated", "target", "target-canary-2", "oci", "", nil, "ready", "running", 1, 70, "r")
	selected = list()
	require.Empty(t, selected["canary/caller-canary-oci"].DeploymentID, "recreating the slug must not revive the binding")

	require.NoError(t, NewQueries(tx).DeleteEnvironmentById(t.Context(), "caller-prod"))
	require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM app_bindings`).Scan(&remaining))
	require.Equal(t, 8, remaining, "deleting a caller environment removes its bindings")
	selected = list()
	require.Len(t, selected, 8)

	exec(`INSERT INTO app_bindings (id,workspace_id,project_id,app_id,environment_id,resource_type,resource_id,name,selection_mode,created_at)
		VALUES ('queue','ws','project','caller','caller-canary','queue','target','jobs','automatic',1),
		('vault','ws','project','caller','caller-canary','vault','target','secrets',NULL,1)`)
	require.Equal(t, selected, list(), "non-app resources must not enter app discovery, even with a matching app ID")
	bindings, err := NewQueries(tx).ListAppBindingsByApp(t.Context(), ListAppBindingsByAppParams{
		WorkspaceID: "ws", ProjectID: "project", AppID: "caller", EnvironmentID: "caller-canary",
	})
	require.NoError(t, err)
	ids := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		ids = append(ids, binding.ID)
	}
	require.ElementsMatch(t, []string{"canary", "canary-auto"}, ids, "only app bindings inject private DNS hostnames")

	require.NoError(t, NewQueries(tx).DeleteAppById(t.Context(), "target"))
	require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM app_bindings WHERE resource_type = 'app' AND resource_id = 'target'`).Scan(&remaining))
	require.Zero(t, remaining)
	require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM app_bindings WHERE resource_id = 'target'`).Scan(&remaining))
	require.Equal(t, 2, remaining, "deleting an app must preserve other resource types with the same ID")
	require.NoError(t, NewQueries(tx).DeleteAppById(t.Context(), "caller"))
	require.NoError(t, tx.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM app_bindings`).Scan(&remaining))
	require.Zero(t, remaining, "deleting the caller removes bindings of every resource type")
}
