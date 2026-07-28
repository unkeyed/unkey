package cluster

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// heartbeatTestHarness exercises the heartbeat handler against a real MySQL schema.
type heartbeatTestHarness struct {
	database db.Database
	service  *Service
	bearer   string
}

// newHeartbeatTestHarness creates the database and authenticated service used by heartbeat tests.
func newHeartbeatTestHarness(t *testing.T) *heartbeatTestHarness {
	t.Helper()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	const bearer = "test-bearer"
	return &heartbeatTestHarness{
		database: database,
		service:  &Service{db: database, bearer: bearer},
		bearer:   bearer,
	}
}

// heartbeat sends one authenticated heartbeat and requires it to succeed.
func (h *heartbeatTestHarness) heartbeat(t *testing.T, platform, region string) {
	t.Helper()

	req := connect.NewRequest(&ctrlv1.HeartbeatRequest{
		Region: &ctrlv1.RegionKey{Platform: platform, Name: region},
	})
	req.Header().Set("Authorization", "Bearer "+h.bearer)
	_, err := h.service.Heartbeat(t.Context(), req)
	require.NoError(t, err)
}

// readCluster loads the cluster that owns regionID.
func (h *heartbeatTestHarness) readCluster(t *testing.T, regionID string) db.Cluster {
	t.Helper()

	var cluster db.Cluster
	err := h.database.RW().QueryRowContext(t.Context(), `
			SELECT pk, id, region_id, platform, region, state, last_heartbeat_at
			FROM clusters
			WHERE region_id = ?
		`, regionID).Scan(
		&cluster.Pk,
		&cluster.ID,
		&cluster.RegionID,
		&cluster.Platform,
		&cluster.Region,
		&cluster.State,
		&cluster.LastHeartbeatAt,
	)
	require.NoError(t, err)
	return cluster
}

// TestHeartbeatPopulatesClusterRegionMetadata guarantees that the transitional
// cluster metadata mirrors region identity without allowing heartbeats to
// reactivate a cluster that operators disabled.
func TestHeartbeatPopulatesClusterRegionMetadata(t *testing.T) {
	h := newHeartbeatTestHarness(t)
	platform := uid.New("platform")

	t.Run("refreshes metadata without changing cluster identity or state", func(t *testing.T) {
		regionName := uid.New("region")
		h.heartbeat(t, platform, regionName)

		region, err := h.database.FindRegionByPlatformAndName(t.Context(), db.FindRegionByPlatformAndNameParams{
			Platform: platform,
			Name:     regionName,
		})
		require.NoError(t, err)

		original := h.readCluster(t, region.ID)
		require.Equal(t, platform, original.Platform)
		require.Equal(t, regionName, original.Region)
		require.Equal(t, db.ClustersStateActive, original.State)
		require.NotZero(t, original.LastHeartbeatAt)

		_, err = h.database.RW().ExecContext(t.Context(), `
			UPDATE clusters
			SET platform = '', region = '', state = 'disabled', last_heartbeat_at = 1
			WHERE region_id = ?
		`, region.ID)
		require.NoError(t, err)

		h.heartbeat(t, platform, regionName)

		refreshed := h.readCluster(t, region.ID)
		require.Equal(t, original.ID, refreshed.ID)
		require.Equal(t, platform, refreshed.Platform)
		require.Equal(t, regionName, refreshed.Region)
		require.Equal(t, db.ClustersStateDisabled, refreshed.State)
		require.Greater(t, refreshed.LastHeartbeatAt, uint64(1))
	})

	t.Run("creates a disabled cluster for an unschedulable region", func(t *testing.T) {
		regionID := uid.New(uid.RegionPrefix)
		disabledRegionName := uid.New("region")
		err := h.database.UpsertRegion(t.Context(), db.UpsertRegionParams{
			ID:       regionID,
			Name:     disabledRegionName,
			Platform: platform,
		})
		require.NoError(t, err)
		_, err = h.database.RW().ExecContext(t.Context(), "UPDATE regions SET can_schedule = false WHERE id = ?", regionID)
		require.NoError(t, err)

		h.heartbeat(t, platform, disabledRegionName)

		cluster := h.readCluster(t, regionID)
		require.Equal(t, platform, cluster.Platform)
		require.Equal(t, disabledRegionName, cluster.Region)
		require.Equal(t, db.ClustersStateDisabled, cluster.State)
	})
}

// TestFindClusterRegionByPlatformAndNameUsesClusterMetadata guarantees agent
// region resolution no longer depends on the legacy region catalog values.
func TestFindClusterRegionByPlatformAndNameUsesClusterMetadata(t *testing.T) {
	h := newHeartbeatTestHarness(t)
	platform := uid.New("platform")
	regionName := uid.New("region")
	h.heartbeat(t, platform, regionName)

	_, err := h.database.RW().ExecContext(t.Context(), `
		UPDATE regions
		SET name = ?, platform = ?
		WHERE name = ? AND platform = ?
	`, uid.New("legacy-region"), uid.New("legacy-platform"), regionName, platform)
	require.NoError(t, err)

	region, err := h.database.FindClusterRegionByPlatformAndName(t.Context(), db.FindClusterRegionByPlatformAndNameParams{
		Platform: platform,
		Name:     regionName,
	})
	require.NoError(t, err)
	require.Equal(t, regionName, region.Name)
	require.Equal(t, platform, region.Platform)

	regions, err := h.database.ListRegions(t.Context())
	require.NoError(t, err)
	regionsByID := make(map[string]db.ListRegionsRow, len(regions))
	for _, listedRegion := range regions {
		regionsByID[listedRegion.ID] = listedRegion
	}
	listedRegion, ok := regionsByID[region.ID]
	require.True(t, ok)
	require.Equal(t, regionName, listedRegion.Name)
	require.Equal(t, platform, listedRegion.Platform)
	require.True(t, listedRegion.CanSchedule)
}
