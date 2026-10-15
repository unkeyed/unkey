package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeletionFindsDescendantsWithMissingParents(t *testing.T) {
	mysql := containers.MySQLIsolated(t)
	database, err := db.New(mysql.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	ctx := t.Context()
	for _, statement := range []string{
		`INSERT INTO apps (id, workspace_id, project_id, name, slug, created_at)
		 VALUES ('app', 'ws', 'project', 'app', 'app', 1)`,
		`INSERT INTO environments (id, workspace_id, project_id, app_id, slug, created_at)
		 VALUES ('env', 'ws', 'project', 'app', 'env', 1), ('env-orphan', 'ws', 'project', 'missing-app', 'env', 1)`,
		`INSERT INTO deployments (id, k8s_name, workspace_id, project_id, app_id, environment_id, sentinel_config, cpu_millicores, memory_mib, encrypted_environment_variables, created_at)
		 VALUES ('live', 'live', 'ws', 'project', 'app', 'env', '', 250, 256, '', 1),
		 ('orphan', 'orphan', 'ws', 'project', 'app', 'missing-env', '', 250, 256, '', 1),
		 ('deep-orphan', 'deep-orphan', 'ws', 'project', 'missing-both', 'missing-both', '', 250, 256, '', 1),
		 ('sibling', 'sibling', 'ws', 'other-project', 'other-app', 'other-env', '', 250, 256, '', 1)`,
	} {
		_, err := database.RW().ExecContext(ctx, statement)
		require.NoError(t, err)
	}
	apps, err := database.ListAppIdsByProject(ctx, db.ListAppIdsByProjectParams{ProjectID: "project"})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"app", "missing-app", "missing-both"}, apps)
	environments, err := database.ListEnvironmentIdsByApp(ctx, db.ListEnvironmentIdsByAppParams{AppID: "app"})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"env", "missing-env"}, environments)
	environments, err = database.ListEnvironmentIdsByApp(ctx, db.ListEnvironmentIdsByAppParams{AppID: "missing-both"})
	require.NoError(t, err)
	require.Equal(t, []string{"missing-both"}, environments)
}
