package cluster

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestFullSyncSelectsOnlyOutstandingDeploymentWork(t *testing.T) {
	t.Run("settled stopped topology", func(t *testing.T) {
		f := newDeletionFixture(t, 1)
		stopDeploymentForSyncTest(t, f, true)
		require.Empty(t, listSyncRows(t, f))
	})

	t.Run("stopped topology with instance", func(t *testing.T) {
		f := newDeletionFixture(t, 1)
		stopDeploymentForSyncTest(t, f, true)
		f.createInstance(t, f.regions[0])
		require.Len(t, listSyncRows(t, f), 1)
	})

	t.Run("stopped topology with stale deployment status", func(t *testing.T) {
		f := newDeletionFixture(t, 1)
		stopDeploymentForSyncTest(t, f, false)
		require.Len(t, listSyncRows(t, f), 1)
	})

	t.Run("stopped topology with deleting parent", func(t *testing.T) {
		f := newDeletionFixture(t, 1)
		stopDeploymentForSyncTest(t, f, true)
		require.NoError(t, f.database.MarkEnvironmentDeleting(t.Context(), db.MarkEnvironmentDeletingParams{
			ID: f.environmentID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		}))
		rows := listSyncRows(t, f)
		require.Len(t, rows, 1)
		require.NotZero(t, rows[0].RemovalRequired)
	})
}

func TestStoppedTopologyStatusRepairIsRegional(t *testing.T) {
	f := newDeletionFixture(t, 2)
	stopDeploymentForSyncTest(t, f, true)
	f.createInstance(t, f.regions[1])
	for i, region := range f.regions {
		row, err := f.database.FindDeploymentTopologyByDeploymentAndRegion(t.Context(), db.FindDeploymentTopologyByDeploymentAndRegionParams{
			DeploymentID: f.deployment.ID, RegionID: region.ID,
		})
		require.NoError(t, err)
		require.Equal(t, int64(i), row.StatusRepairRequired)
	}
	_, err := f.database.RW().ExecContext(t.Context(), "UPDATE deployments SET status = 'ready' WHERE id = ?", f.deployment.ID)
	require.NoError(t, err)
	row, err := f.database.FindDeploymentTopologyByDeploymentAndRegion(t.Context(), db.FindDeploymentTopologyByDeploymentAndRegionParams{
		DeploymentID: f.deployment.ID, RegionID: f.regions[0].ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), row.StatusRepairRequired)
}

func TestStoppingTopologyAlwaysAdvancesItsChangeTimestamp(t *testing.T) {
	f := newDeletionFixture(t, 1)
	for i, timestamp := range []int64{1000, 1000, 900} {
		require.NoError(t, f.database.StopDeploymentTopologiesByEnvironment(t.Context(), db.StopDeploymentTopologiesByEnvironmentParams{
			EnvironmentID: f.environmentID, UpdatedAt: sql.NullInt64{Int64: timestamp, Valid: true},
		}))
		var updatedAt int64
		require.NoError(t, f.database.RW().QueryRowContext(t.Context(), "SELECT updated_at FROM deployment_topology WHERE deployment_id = ?", f.deployment.ID).Scan(&updatedAt))
		require.Equal(t, int64(1000+i), updatedAt)
	}
}

func TestActiveParentGuardsShareLocksAndBlockDeletion(t *testing.T) {
	f := newDeletionFixture(t, 1)
	first, err := f.database.RW().Begin(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() {
		err := first.Rollback()
		if err != sql.ErrTxDone {
			require.NoError(t, err)
		}
	})
	_, err = db.NewQueries(first).LockActiveEnvironment(t.Context(), f.environmentID)
	require.NoError(t, err)

	second, err := f.database.RW().Begin(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() {
		err := second.Rollback()
		if err != sql.ErrTxDone {
			require.NoError(t, err)
		}
	})
	secondCtx, cancelSecond := context.WithTimeout(t.Context(), time.Second)
	defer cancelSecond()
	_, err = db.NewQueries(second).LockActiveEnvironment(secondCtx, f.environmentID)
	require.NoError(t, err)

	deleteCtx, cancelDelete := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancelDelete()
	err = f.database.MarkEnvironmentDeleting(deleteCtx, db.MarkEnvironmentDeletingParams{
		ID: f.environmentID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, second.Commit())
	require.NoError(t, first.Commit())
}

func stopDeploymentForSyncTest(t *testing.T, f deletionFixture, stopStatus bool) {
	t.Helper()
	now := sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}
	require.NoError(t, f.database.StopDeploymentTopologiesByEnvironment(t.Context(), db.StopDeploymentTopologiesByEnvironmentParams{
		EnvironmentID: f.environmentID, UpdatedAt: now,
	}))
	require.NoError(t, f.database.StopDeploymentsByEnvironment(t.Context(), db.StopDeploymentsByEnvironmentParams{
		EnvironmentID: f.environmentID, UpdatedAt: now,
	}))
	if stopStatus {
		require.NoError(t, f.database.StopDeploymentIfNoInstances(t.Context(), db.StopDeploymentIfNoInstancesParams{
			ID: f.deployment.ID, UpdatedAt: now,
		}))
	}
}

func listSyncRows(t *testing.T, f deletionFixture) []db.ListAllDeploymentTopologiesByRegionRow {
	t.Helper()
	rows, err := f.database.ListAllDeploymentTopologiesByRegion(t.Context(), db.ListAllDeploymentTopologiesByRegionParams{
		RegionID: f.regions[0].ID, Limit: 10,
	})
	require.NoError(t, err)
	return rows
}
