package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
)

func TestRunningSnapshotsRetainPinsAfterDefaultsChange(t *testing.T) {
	database := openTestDatabase(t)
	for _, change := range []string{"retargeted", "deleted"} {
		t.Run(change, func(t *testing.T) {
			q := NewQueries(beginRollbackTx(t, database))
			f := seedAppConnections(t, q)
			switch change {
			case "retargeted":
				require.NoError(t, q.UpdateConnectionAppTarget(t.Context(), UpdateConnectionAppTargetParams{
					ConnectionID: f.connections["pinned"], SelectionMode: ConnectionAppTargetsSelectionModeAutomatic,
				}))
			case "deleted":
				deleted, err := q.DeleteAppConnectionById(t.Context(), f.connections["pinned"])
				require.NoError(t, err)
				require.Equal(t, int64(2), deleted, "the default and its app target are deleted together")
			}
			pin := func() bool {
				t.Helper()
				pinned, err := q.ExistsAppConnectionPinningDeployment(t.Context(), ExistsAppConnectionPinningDeploymentParams{
					DeploymentID: sql.NullString{String: f.deployments["target-pinned-stopped"], Valid: true},
				})
				require.NoError(t, err)
				return pinned
			}
			for _, status := range []mysqltype.DeploymentsStatus{
				mysqltype.DeploymentsStatusPending, mysqltype.DeploymentsStatusStarting, mysqltype.DeploymentsStatusBuilding,
				mysqltype.DeploymentsStatusDeploying, mysqltype.DeploymentsStatusNetwork, mysqltype.DeploymentsStatusFinalizing,
				mysqltype.DeploymentsStatusReady,
			} {
				setTestDeploymentStatus(t, q, f.deployments["caller-pin"], status, sql.NullInt64{})
				require.True(t, pin(), status)
			}
			for _, status := range []mysqltype.DeploymentsStatus{
				mysqltype.DeploymentsStatusAwaitingApproval, mysqltype.DeploymentsStatusFailed, mysqltype.DeploymentsStatusCancelled,
				mysqltype.DeploymentsStatusSuperseded, mysqltype.DeploymentsStatusStopped, mysqltype.DeploymentsStatusSkipped,
			} {
				setTestDeploymentStatus(t, q, f.deployments["caller-pin"], status, sql.NullInt64{})
				require.False(t, pin(), status)
			}
			setTestDeploymentStatus(t, q, f.deployments["caller-pin"], mysqltype.DeploymentsStatusReady, sql.NullInt64{})
			setTestDesiredState(t, q, f.deployments["caller-pin"], mysqltype.DeploymentsDesiredStateStopped)
			require.False(t, pin())
			setTestDesiredState(t, q, f.deployments["caller-pin"], mysqltype.DeploymentsDesiredStateRunning)
			require.True(t, pin())
		})
	}
}
