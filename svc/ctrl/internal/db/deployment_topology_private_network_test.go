package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

func TestDeploymentTopologyPrivateNetworkFollowsStoredDecision(t *testing.T) {
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

	exec(`INSERT INTO workspaces (id,org_id,name,slug,k8s_namespace,beta_features) VALUES ('pn-ws','pn-org','Workspace','pn-topology','pn-namespace','{}')`)
	exec(`INSERT INTO projects (id,workspace_id,name,slug,created_at) VALUES ('pn-project','pn-ws','Project','project',1)`)
	exec(`INSERT INTO apps (id,workspace_id,project_id,name,slug,source_type,created_at) VALUES
		('pn-api','pn-ws','pn-project','API','api','git',1),('pn-db','pn-ws','pn-project','DB','db','git',1)`)
	exec(`INSERT INTO environments (id,workspace_id,project_id,app_id,slug,kind,created_at) VALUES ('pn-env','pn-ws','pn-project','pn-api','production','production',1)`)
	exec(`INSERT INTO regions (id,name,platform) VALUES ('pn-region','pn-region','pn-platform')`)
	for _, deployment := range []struct {
		id      string
		enabled bool
	}{{id: "pn-enabled", enabled: true}, {id: "pn-disabled", enabled: false}} {
		exec(`INSERT INTO deployments (id,k8s_name,workspace_id,project_id,environment_id,app_id,sentinel_config,cpu_millicores,memory_mib,desired_state,encrypted_environment_variables,status,created_at,private_networking)
			VALUES (?,?,'pn-ws','pn-project','pn-env','pn-api','{}',100,128,'running','{}','ready',1,?)`, deployment.id, deployment.id, deployment.enabled)
		exec(`INSERT INTO deployment_topology (workspace_id,deployment_id,region_id,desired_status,created_at) VALUES ('pn-ws',?,'pn-region','running',1)`, deployment.id)
	}

	enrolled := func() map[string]bool {
		t.Helper()
		byDeployment := map[string]bool{}
		rows, listErr := NewQueries(tx).ListAllDeploymentTopologiesByRegion(t.Context(), ListAllDeploymentTopologiesByRegionParams{RegionID: "pn-region", AfterPk: 0, Limit: 10})
		require.NoError(t, listErr)
		for _, row := range rows {
			byDeployment[row.DeploymentID] = row.PrivateNetworkEnrolled
			found, findErr := NewQueries(tx).FindDeploymentTopologyByDeploymentAndRegion(t.Context(), FindDeploymentTopologyByDeploymentAndRegionParams{DeploymentID: row.DeploymentID, RegionID: "pn-region"})
			require.NoError(t, findErr)
			require.Equal(t, row.PrivateNetworkEnrolled, found.PrivateNetworkEnrolled, "both topology reads agree for %s", row.DeploymentID)
		}
		return byDeployment
	}

	want := map[string]bool{"pn-enabled": true, "pn-disabled": false}
	require.Equal(t, want, enrolled(), "without bindings")

	exec(`INSERT INTO app_bindings (id,workspace_id,project_id,app_id,environment_id,resource_type,resource_id,name,selection_mode,created_at)
		VALUES ('pn-binding','pn-ws','pn-project','pn-api','pn-env','app','pn-db','database','automatic',1)`)
	require.Equal(t, want, enrolled(), "adding a binding changes no running deployment")

	exec(`DELETE FROM app_bindings WHERE id = 'pn-binding'`)
	require.Equal(t, want, enrolled(), "deleting every binding changes no running deployment")
}
