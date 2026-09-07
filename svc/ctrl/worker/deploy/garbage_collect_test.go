package deploy

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploymentretention"
)

func TestDeploymentRecovery(t *testing.T) {
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	w := &Workflow{db: database}
	now := time.Now().Truncate(time.Millisecond)

	for _, restore := range []bool{false, true} {
		t.Run(map[bool]string{false: "purge", true: "restore"}[restore], func(t *testing.T) {
			id := seedGCDeployment(t, database, now, "preview", "stopped")
			ctx := t.Context()
			candidatesAt := func(at time.Time) []string {
				t.Helper()
				production, preview := deploymentretention.Cutoffs(at)
				rows, err := database.ListDeploymentGCCandidates(ctx, db.ListDeploymentGCCandidatesParams{
					ProductionCutoff: production,
					PreviewCutoff:    preview,
					RecoveryCutoff:   sql.NullInt64{Int64: at.Add(-deploymentretention.RecoveryAge).UnixMilli(), Valid: true},
					KeepSuccessful:   deploymentretention.Successful,
					Limit:            500,
				})
				require.NoError(t, err)
				var ids []string
				for _, row := range rows {
					ids = append(ids, row.ID)
				}
				return ids
			}
			require.Contains(t, candidatesAt(now), id)
			_, err := database.RW().ExecContext(ctx, `INSERT INTO deployment_steps
				(workspace_id, project_id, app_id, environment_id, deployment_id, started_at)
				SELECT workspace_id, project_id, app_id, environment_id, id, created_at FROM deployments WHERE id = ?`, id)
			require.NoError(t, err)
			image := sql.NullString{String: "registry.depot.dev/test:proj_gc-" + id, Valid: true}
			_, err = database.RW().ExecContext(ctx, "UPDATE deployments SET image = ? WHERE id = ?", image.String, id)
			require.NoError(t, err)

			deleted, err := w.collectDeployment(ctx, id, now)
			require.NoError(t, err)
			require.False(t, deleted)
			_, err = database.FindDeploymentById(ctx, id)
			require.True(t, db.IsNotFound(err), "normal lookups must hide recoverable deployments")
			_, err = database.FindDeploymentWithEnvironmentAndApp(ctx, id)
			require.True(t, db.IsNotFound(err), "rollback lookup must hide recoverable deployments")
			exists, err := database.DeploymentExistsIncludingDeleted(ctx, id)
			require.NoError(t, err)
			require.True(t, exists)
			referenced, err := database.DeploymentImageExistsIncludingDeleted(ctx, image)
			require.NoError(t, err)
			require.True(t, referenced)
			var steps int
			require.NoError(t, database.RW().QueryRowContext(ctx, "SELECT COUNT(*) FROM deployment_steps WHERE deployment_id = ?", id).Scan(&steps))
			require.Equal(t, 1, steps)

			beforeExpiry := now.Add(deploymentretention.RecoveryAge - time.Millisecond)
			require.NotContains(t, candidatesAt(beforeExpiry), id)
			deleted, err = w.collectDeployment(ctx, id, beforeExpiry)
			require.NoError(t, err)
			require.False(t, deleted)
			var deletedAt int64
			require.NoError(t, database.RW().QueryRowContext(ctx, "SELECT deleted_at FROM deployments WHERE id = ?", id).Scan(&deletedAt))
			require.Equal(t, now.UnixMilli(), deletedAt, "repeated GC must not restart recovery")

			restoreTime := now.Add(deploymentretention.RecoveryAge)
			require.Contains(t, candidatesAt(restoreTime), id)
			if restore {
				restoreTime = beforeExpiry
			}
			rows, err := database.RestoreDeployment(ctx, db.RestoreDeploymentParams{
				DeploymentID:   id,
				RestoredAt:     sql.NullInt64{Int64: restoreTime.UnixMilli(), Valid: true},
				RecoveryCutoff: sql.NullInt64{Int64: restoreTime.Add(-deploymentretention.RecoveryAge).UnixMilli(), Valid: true},
			})
			require.NoError(t, err)
			if restore {
				require.Equal(t, int64(1), rows)
				deployment, err := database.FindDeploymentById(ctx, id)
				require.NoError(t, err)
				require.Equal(t, "stopped", string(deployment.DesiredState))
				require.Equal(t, "stopped", string(deployment.Status))
				deleted, err = w.collectDeployment(ctx, id, restoreTime)
				require.NoError(t, err)
				require.False(t, deleted)
				_, err = database.FindDeploymentById(ctx, id)
				require.NoError(t, err, "restored deployment gets a fresh retention window")
				return
			}
			require.Zero(t, rows, "restoration must fail at the exact recovery boundary")
			deleted, err = w.collectDeployment(ctx, id, restoreTime)
			require.NoError(t, err)
			require.True(t, deleted)
			exists, err = database.DeploymentExistsIncludingDeleted(ctx, id)
			require.NoError(t, err)
			require.False(t, exists)
			require.NoError(t, database.RW().QueryRowContext(ctx, "SELECT COUNT(*) FROM deployment_steps WHERE deployment_id = ?", id).Scan(&steps))
			require.Zero(t, steps)
			referenced, err = database.DeploymentImageExistsIncludingDeleted(ctx, image)
			require.NoError(t, err)
			require.False(t, referenced)
		})
	}
}

func TestGCRechecksProtection(t *testing.T) {
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	w := &Workflow{db: database}
	now := time.Now().Truncate(time.Millisecond)
	for _, protection := range []string{"current", "sticky", "waking", "production_success"} {
		t.Run(protection, func(t *testing.T) {
			kind := "preview"
			if protection == "production_success" {
				kind = "production"
			}
			id := seedGCDeployment(t, database, now, kind, "stopped")
			ctx := t.Context()
			switch protection {
			case "current":
				_, err = database.RW().ExecContext(ctx, "UPDATE apps SET current_deployment_id = ? WHERE id = (SELECT app_id FROM deployments WHERE id = ?)", id, id)
			case "sticky":
				_, err = database.RW().ExecContext(ctx, `INSERT INTO frontline_routes
					(id, project_id, app_id, environment_id, deployment_id, fully_qualified_domain_name, sticky, created_at)
					SELECT id, project_id, app_id, environment_id, id, CONCAT(id, '.test'), 'branch', created_at FROM deployments WHERE id = ?`, id)
			case "waking":
				_, err = database.RW().ExecContext(ctx, "UPDATE deployments SET desired_state = 'running' WHERE id = ?", id)
			case "production_success":
			}
			require.NoError(t, err)
			deleted, err := w.collectDeployment(ctx, id, now)
			require.NoError(t, err)
			require.False(t, deleted)
			_, err = database.FindDeploymentById(ctx, id)
			require.NoError(t, err)
		})
	}
}

func seedGCDeployment(t *testing.T, database db.Database, now time.Time, kind, status string) string {
	t.Helper()
	id := uid.New(uid.DeploymentPrefix)
	appID := uid.New(uid.AppPrefix)
	envID := uid.New(uid.EnvironmentPrefix)
	projectID := uid.New(uid.ProjectPrefix)
	workspaceID := uid.New(uid.WorkspacePrefix)
	ctx := t.Context()
	_, err := database.RW().ExecContext(ctx, "INSERT INTO apps (id, workspace_id, project_id, name, slug, created_at) VALUES (?, ?, ?, 'gc', 'gc', ?)", appID, workspaceID, projectID, now.UnixMilli())
	require.NoError(t, err)
	_, err = database.RW().ExecContext(ctx, "INSERT INTO environments (id, workspace_id, project_id, app_id, slug, kind, created_at) VALUES (?, ?, ?, ?, 'gc', ?, ?)", envID, workspaceID, projectID, appID, kind, now.UnixMilli())
	require.NoError(t, err)
	_, err = database.RW().ExecContext(ctx, `INSERT INTO deployments
		(id, k8s_name, workspace_id, project_id, app_id, environment_id, sentinel_config, encrypted_environment_variables, cpu_millicores, memory_mib, desired_state, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, '', '', 100, 128, 'stopped', ?, ?)`, id, id, workspaceID, projectID, appID, envID, status, now.Add(-90*24*time.Hour).UnixMilli())
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx := context.Background()
		require.NoError(t, deleteDeploymentChildren(ctx, db.NewQueries(database.RW()), id))
		_, err := database.DeleteDeploymentByIDForGC(ctx, id)
		require.NoError(t, err)
		_, err = database.RW().ExecContext(ctx, "DELETE FROM environments WHERE id = ?", envID)
		require.NoError(t, err)
		_, err = database.RW().ExecContext(ctx, "DELETE FROM apps WHERE id = ?", appID)
		require.NoError(t, err)
	})
	return id
}
