package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeploymentStatusPreservesFirstReadyAt(t *testing.T) {
	server := containers.MySQL(t)
	database, err := sql.Open("mysql", server.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	for _, tt := range []struct {
		name       string
		status     mysqltype.DeploymentsStatus
		updatedAt  sql.NullInt64
		firstWake  sql.NullInt64
		firstReady int64
	}{
		{name: "never ready", status: mysqltype.DeploymentsStatusPending, firstReady: 200},
		{name: "legacy ready", status: mysqltype.DeploymentsStatusReady, updatedAt: sql.NullInt64{Int64: 80, Valid: true}, firstWake: sql.NullInt64{Int64: 80, Valid: true}, firstReady: 80},
		{name: "legacy stopped", status: mysqltype.DeploymentsStatusStopped, updatedAt: sql.NullInt64{Int64: 90, Valid: true}, firstWake: sql.NullInt64{Int64: 90, Valid: true}, firstReady: 90},
		{name: "legacy without update time", status: mysqltype.DeploymentsStatusStopped, firstWake: sql.NullInt64{Int64: 10, Valid: true}, firstReady: 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := database.BeginTx(t.Context(), nil)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tx.Rollback()) })

			id := uid.New("dep")
			_, err = tx.ExecContext(t.Context(), `INSERT INTO deployments
(id,k8s_name,workspace_id,project_id,app_id,environment_id,sentinel_config,cpu_millicores,memory_mib,encrypted_environment_variables,status,created_at,updated_at)
VALUES (?,?,'ws','project','app','preview','{}',100,128,'{}',?,10,?)`, id, id, tt.status, tt.updatedAt)
			require.NoError(t, err)

			q := NewQueries(tx)
			update := func(status mysqltype.DeploymentsStatus, timestamp int64, expected sql.NullInt64) {
				t.Helper()
				require.NoError(t, q.UpdateDeploymentStatus(t.Context(), UpdateDeploymentStatusParams{
					ID: id, Status: status, UpdatedAt: sql.NullInt64{Int64: timestamp, Valid: true},
				}))
				var actual sql.NullInt64
				require.NoError(t, tx.QueryRowContext(t.Context(), "SELECT first_ready_at FROM deployments WHERE id = ?", id).Scan(&actual))
				require.Equal(t, expected, actual)
				deployment, err := q.FindDeploymentById(t.Context(), id)
				require.NoError(t, err)
				require.Equal(t, status, deployment.Status)
				require.Equal(t, timestamp, deployment.UpdatedAt.Int64)
			}

			update(mysqltype.DeploymentsStatusDeploying, 100, tt.firstWake)
			update(mysqltype.DeploymentsStatusFailed, 150, tt.firstWake)
			readyAt := sql.NullInt64{Int64: tt.firstReady, Valid: true}
			update(mysqltype.DeploymentsStatusReady, 200, readyAt)
			update(mysqltype.DeploymentsStatusReady, 300, readyAt)
			update(mysqltype.DeploymentsStatusStopped, 400, readyAt)
			update(mysqltype.DeploymentsStatusDeploying, 500, readyAt)
			update(mysqltype.DeploymentsStatusReady, 600, readyAt)
		})
	}
}
