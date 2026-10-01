package deploymentstream

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cdc"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type topologyRow struct {
	deploymentID string
	regionID     string
}

func TestWatch_VitessDeletesFilteringAndResume(t *testing.T) {
	for _, test := range []struct {
		name     string
		expected []string
		remove   func(ctx context.Context, database db.Database, rows []topologyRow) error
	}{
		{name: "soft delete", expected: []string{"included", "historical", "included", "historical"}, remove: func(ctx context.Context, database db.Database, rows []topologyRow) error {
			for _, row := range rows {
				err := database.UpdateDeploymentTopologyDesiredStatus(ctx, db.UpdateDeploymentTopologyDesiredStatusParams{
					DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
					UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
					DeploymentID:  row.deploymentID,
					RegionID:      row.regionID,
				})
				if err != nil {
					return err
				}
			}
			return nil
		}},
		{name: "hard delete", expected: []string{"included", "historical", "included", "historical"}, remove: func(ctx context.Context, database db.Database, rows []topologyRow) error {
			_, err := database.DeleteDeploymentTopologiesByRegionIds(ctx, db.DeleteDeploymentTopologiesByRegionIdsParams{
				RegionIds: []string{rows[0].regionID, rows[1].regionID},
				Limit:     math.MaxInt32,
			})
			return err
		}},
		{name: "touch stopped", expected: []string{"included", "historical", "historical"}, remove: func(ctx context.Context, database db.Database, rows []topologyRow) error {
			return database.UpdateDeploymentTopologyDesiredStatus(ctx, db.UpdateDeploymentTopologyDesiredStatusParams{
				DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: 2},
				DeploymentID:  rows[2].deploymentID,
				RegionID:      rows[2].regionID,
			})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			vitess := containers.Vitess(t)
			database, err := db.New(vitess.DSN, sqlcomment.Disabled())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.Close()) })
			region := uid.New(uid.RegionPrefix)
			otherRegion := region + "_other"
			t.Cleanup(func() {
				_, err := database.DeleteDeploymentTopologiesByRegionIds(context.Background(), db.DeleteDeploymentTopologiesByRegionIdsParams{
					RegionIds: []string{region, otherRegion},
					Limit:     math.MaxInt32,
				})
				require.NoError(t, err)
			})
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			workspaceID := uid.New(uid.WorkspacePrefix)
			included := uid.New(uid.DeploymentPrefix)
			excluded := uid.New(uid.DeploymentPrefix)
			historical := uid.New(uid.DeploymentPrefix)
			ids := map[string]string{"included": included, "historical": historical}
			expected := make([]string, 0, len(test.expected))
			for _, name := range test.expected {
				expected = append(expected, ids[name])
			}
			require.NoError(t, database.Bulk().InsertDeploymentTopologies(ctx, []db.InsertDeploymentTopologyParams{
				topology(workspaceID, included, region, db.DeploymentTopologyDesiredStatusRunning, 1),
				topology(workspaceID, excluded, otherRegion, db.DeploymentTopologyDesiredStatusRunning, 1),
				topology(workspaceID, historical, region, db.DeploymentTopologyDesiredStatusStopped, 1),
			}))
			client, err := New(cdc.Config{Address: vitess.Address, Keyspace: "unkey", Insecure: true})
			require.NoError(t, err)
			var delivered []string
			var token []byte
			updated := false
			err = client.Watch(ctx, region, nil, func(event Event) error {
				if event.DeploymentID != "" {
					require.Empty(t, event.ResumeToken)
					require.Equal(t, updated, event.HasBefore)
					delivered = append(delivered, event.DeploymentID)
					return nil
				}
				require.NotEmpty(t, event.ResumeToken)
				token = event.ResumeToken
				if len(delivered) == 2 && !updated {
					updated = true
					return test.remove(ctx, database, []topologyRow{
						{deploymentID: included, regionID: region},
						{deploymentID: excluded, regionID: otherRegion},
						{deploymentID: historical, regionID: region},
					})
				}
				if len(delivered) >= len(test.expected) {
					cancel()
				}
				return nil
			})
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, expected, delivered, "stopped-row updates must be delivered while other regions stay excluded")
			require.NotEmpty(t, token)

			resumeCtx, resumeCancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer resumeCancel()
			err = client.Watch(resumeCtx, otherRegion, token, func(Event) error { return errors.New("unexpected cross-region event") })
			require.ErrorIs(t, err, cdc.ErrInvalidToken)
			whileOffline := uid.New(uid.DeploymentPrefix)
			require.NoError(t, database.InsertDeploymentTopology(resumeCtx,
				topology(workspaceID, whileOffline, region, db.DeploymentTopologyDesiredStatusStopped, 2)))
			require.NoError(t, database.UpdateDeploymentTopologyDesiredStatus(resumeCtx, db.UpdateDeploymentTopologyDesiredStatusParams{
				DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: 3},
				DeploymentID:  whileOffline,
				RegionID:      region,
			}))
			delivered = nil
			err = client.Watch(resumeCtx, region, token, func(event Event) error {
				if event.DeploymentID != "" {
					require.Equal(t, len(delivered) > 0, event.HasBefore)
					delivered = append(delivered, event.DeploymentID)
					return nil
				}
				if len(delivered) == 2 {
					resumeCancel()
				}
				return nil
			})
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, []string{whileOffline, whileOffline}, delivered, "resume must deliver stopped-row updates without copying existing history")
		})
	}
}

func TestWatch_VitessRetriesFailedDeliveryAndResumesPartialSnapshot(t *testing.T) {
	vitess := containers.Vitess(t)
	database, err := db.New(vitess.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	region := uid.New(uid.RegionPrefix)
	t.Cleanup(func() {
		for {
			deleted, err := database.DeleteDeploymentTopologiesByRegionIds(context.Background(), db.DeleteDeploymentTopologiesByRegionIdsParams{
				RegionIds: []string{region},
				Limit:     5000,
			})
			require.NoError(t, err)
			if deleted == 0 {
				break
			}
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	const total = 20000
	const insertBatchSize = 5000
	workspaceID := uid.New(uid.WorkspacePrefix)
	snapshotIDs := make([]string, 0, total)
	rows := make([]db.InsertDeploymentTopologyParams, 0, total)
	for range total {
		// IDs this long keep the snapshot above one VStream packet, so it is checkpointed mid-copy.
		deploymentID := uid.New(uid.DeploymentPrefix, 16)
		snapshotIDs = append(snapshotIDs, deploymentID)
		rows = append(rows, topology(workspaceID, deploymentID, region, db.DeploymentTopologyDesiredStatusRunning, 1))
	}
	for start := 0; start < total; start += insertBatchSize {
		require.NoError(t, database.Bulk().InsertDeploymentTopologies(ctx, rows[start:min(start+insertBatchSize, total)]))
	}
	client, err := New(cdc.Config{Address: vitess.Address, Keyspace: "unkey", Insecure: true})
	require.NoError(t, err)
	failure := errors.New("apply failed")
	attempts := 0
	err = client.Watch(ctx, region, nil, func(event Event) error {
		if event.DeploymentID == "" {
			if attempts > 0 {
				return errors.New("checkpoint advanced past failed delivery")
			}
			return nil
		}
		attempts++
		if attempts == 2 {
			return failure
		}
		return nil
	})
	require.ErrorIs(t, err, failure)
	require.Equal(t, 2, attempts)
	delivered := make(map[string]int)
	stop := errors.New("disconnect at checkpoint")
	var token []byte
	err = client.Watch(ctx, region, nil, func(event Event) error {
		if event.DeploymentID != "" {
			delivered[event.DeploymentID]++
			return nil
		}
		if len(delivered) == 0 {
			return nil
		}
		token = event.ResumeToken
		return stop
	})
	require.ErrorIs(t, err, stop)
	require.Less(t, len(delivered), total, "disconnect must occur before the snapshot completes")
	err = client.Watch(ctx, region, token, func(event Event) error {
		if event.DeploymentID != "" {
			delivered[event.DeploymentID]++
			return nil
		}
		if len(delivered) == total {
			return stop
		}
		return nil
	})
	require.ErrorIs(t, err, stop)
	for _, deploymentID := range snapshotIDs {
		require.Equal(t, 1, delivered[deploymentID], "resume must neither skip nor recopy snapshot rows")
	}
}

func topology(workspaceID, deploymentID, regionID string, status db.DeploymentTopologyDesiredStatus, createdAt int64) db.InsertDeploymentTopologyParams {
	return db.InsertDeploymentTopologyParams{
		WorkspaceID:                workspaceID,
		DeploymentID:               deploymentID,
		RegionID:                   regionID,
		AutoscalingReplicasMin:     1,
		AutoscalingReplicasMax:     1,
		AutoscalingThresholdCpu:    sql.NullInt16{Valid: false},
		AutoscalingThresholdMemory: sql.NullInt16{Valid: false},
		DesiredStatus:              status,
		CreatedAt:                  createdAt,
	}
}
