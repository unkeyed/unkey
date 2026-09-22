package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestDeploymentTopologyWrites_AdvanceRevision verifies the database contract
// that inserts start at zero and each upsert or status change advances atomically.
func TestDeploymentTopologyWrites_AdvanceRevision(t *testing.T) {
	ctx := t.Context()
	database, err := New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	deploymentID := uid.New("dep")
	regionID := uid.New("region")
	params := InsertDeploymentTopologyParams{
		WorkspaceID:                uid.New("ws"),
		DeploymentID:               deploymentID,
		RegionID:                   regionID,
		AutoscalingReplicasMin:     1,
		AutoscalingReplicasMax:     2,
		AutoscalingThresholdCpu:    sql.NullInt16{},
		AutoscalingThresholdMemory: sql.NullInt16{},
		DesiredStatus:              DeploymentTopologyDesiredStatusRunning,
		CreatedAt:                  time.Now().UnixMilli(),
	}

	require.NoError(t, database.InsertDeploymentTopology(ctx, params))
	require.Equal(t, uint32(0), topologyRevision(t, database, deploymentID, regionID))
	require.NoError(t, database.InsertDeploymentTopology(ctx, params))
	require.Equal(t, uint32(1), topologyRevision(t, database, deploymentID, regionID))
	require.NoError(t, database.UpdateDeploymentTopologyDesiredStatus(ctx, UpdateDeploymentTopologyDesiredStatusParams{
		DesiredStatus: DeploymentTopologyDesiredStatusStopped,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		DeploymentID:  deploymentID,
		RegionID:      regionID,
	}))
	require.Equal(t, uint32(2), topologyRevision(t, database, deploymentID, regionID))
}

func topologyRevision(t *testing.T, database Database, deploymentID, regionID string) uint32 {
	t.Helper()
	var revision uint32
	err := database.RO().QueryRowContext(t.Context(),
		"SELECT revision FROM deployment_topology WHERE deployment_id = ? AND region_id = ?",
		deploymentID, regionID,
	).Scan(&revision)
	require.NoError(t, err)
	return revision
}
