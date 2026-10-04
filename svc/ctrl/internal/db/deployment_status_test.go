package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeploymentStatusPreservesFirstReadyAt(t *testing.T) {
	database := openTestDatabase(t)

	for _, tt := range []struct {
		name       string
		status     mysqltype.DeploymentsStatus
		updatedAt  sql.NullInt64
		firstWake  sql.NullInt64
		firstReady int64
	}{
		{name: "never ready", status: mysqltype.DeploymentsStatusPending, firstReady: 200},
		{name: "ready before tracking", status: mysqltype.DeploymentsStatusReady, updatedAt: sql.NullInt64{Int64: 80, Valid: true}, firstWake: sql.NullInt64{Int64: 80, Valid: true}, firstReady: 80},
		{name: "stopped without ready proof", status: mysqltype.DeploymentsStatusStopped, updatedAt: sql.NullInt64{Int64: 90, Valid: true}, firstReady: 200},
		{name: "ready without update time", status: mysqltype.DeploymentsStatusReady, firstWake: sql.NullInt64{Int64: 10, Valid: true}, firstReady: 10},
		{name: "stopped without update time", status: mysqltype.DeploymentsStatusStopped, firstReady: 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tx := beginRollbackTx(t, database)
			id := uid.New(uid.DeploymentPrefix)
			q := NewQueries(tx)
			insertTestDeployment(t, q, testDeployment{
				ID: id, WorkspaceID: uid.New(uid.WorkspacePrefix), ProjectID: uid.New(uid.ProjectPrefix), AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix),
				Status: tt.status, CreatedAt: 10, UpdatedAt: tt.updatedAt,
			})

			update := func(status mysqltype.DeploymentsStatus, timestamp int64, expected sql.NullInt64) {
				t.Helper()
				require.NoError(t, q.UpdateDeploymentStatus(t.Context(), UpdateDeploymentStatusParams{
					ID: id, Status: status, UpdatedAt: sql.NullInt64{Int64: timestamp, Valid: true},
				}))
				deployment, err := q.FindDeploymentById(t.Context(), id)
				require.NoError(t, err)
				require.Equal(t, expected, deployment.FirstReadyAt)
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

func TestConditionalDeploymentStatusRecordsFirstReadyAt(t *testing.T) {
	database := openTestDatabase(t)

	for _, tt := range []struct {
		name           string
		status         mysqltype.DeploymentsStatus
		next           mysqltype.DeploymentsStatus
		firstReady     sql.NullInt64
		wantFirstReady sql.NullInt64
		wantStatus     mysqltype.DeploymentsStatus
		wantUpdatedAt  int64
	}{
		{
			name: "first successful deployment", status: mysqltype.DeploymentsStatusFinalizing,
			next: mysqltype.DeploymentsStatusReady, wantStatus: mysqltype.DeploymentsStatusReady,
			wantFirstReady: sql.NullInt64{Int64: 200, Valid: true}, wantUpdatedAt: 200,
		},
		{
			name: "failed deployment is not ready", status: mysqltype.DeploymentsStatusBuilding,
			next: mysqltype.DeploymentsStatusFailed, wantStatus: mysqltype.DeploymentsStatusFailed,
			wantUpdatedAt: 200,
		},
		{
			name: "preserve earlier readiness", status: mysqltype.DeploymentsStatusDeploying,
			next: mysqltype.DeploymentsStatusReady, wantStatus: mysqltype.DeploymentsStatusReady,
			firstReady:     sql.NullInt64{Int64: 80, Valid: true},
			wantFirstReady: sql.NullInt64{Int64: 80, Valid: true}, wantUpdatedAt: 200,
		},
		{
			name: "cancellation wins", status: mysqltype.DeploymentsStatusCancelled,
			next: mysqltype.DeploymentsStatusReady, wantStatus: mysqltype.DeploymentsStatusCancelled,
			wantUpdatedAt: 100,
		},
		{
			name: "compensation cannot change ready deployment", status: mysqltype.DeploymentsStatusReady,
			next: mysqltype.DeploymentsStatusFailed, wantStatus: mysqltype.DeploymentsStatusReady,
			firstReady:     sql.NullInt64{Int64: 80, Valid: true},
			wantFirstReady: sql.NullInt64{Int64: 80, Valid: true}, wantUpdatedAt: 100,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tx := beginRollbackTx(t, database)
			id := uid.New(uid.DeploymentPrefix)
			q := NewQueries(tx)
			insertTestDeployment(t, q, testDeployment{
				ID: id, WorkspaceID: uid.New(uid.WorkspacePrefix), ProjectID: uid.New(uid.ProjectPrefix), AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix),
				Status: tt.status, FirstReadyAt: tt.firstReady, CreatedAt: 10, UpdatedAt: sql.NullInt64{Int64: 100, Valid: true},
			})
			before, err := q.FindDeploymentById(t.Context(), id)
			require.NoError(t, err)
			require.Equal(t, tt.status, before.Status)
			require.Equal(t, tt.firstReady, before.FirstReadyAt)
			require.Equal(t, sql.NullInt64{Int64: 100, Valid: true}, before.UpdatedAt)

			require.NoError(t, q.UpdateDeploymentStatusIfActive(t.Context(), UpdateDeploymentStatusIfActiveParams{
				ID: id, Status: tt.next, UpdatedAt: sql.NullInt64{Int64: 200, Valid: true},
				ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
			}))
			deployment, err := q.FindDeploymentById(t.Context(), id)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, deployment.Status)
			require.Equal(t, tt.wantFirstReady, deployment.FirstReadyAt)
			require.Equal(t, sql.NullInt64{Int64: tt.wantUpdatedAt, Valid: true}, deployment.UpdatedAt)
		})
	}
}
