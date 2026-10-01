package deploy

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestReserveTopologies pins that two deployments of one workspace cannot both
// reserve when only one fits, and that a re-run for the same deployment does
// not count its own rows. The seeded deployment is 256 MiB, so a 256 MiB quota
// fits one
func TestReserveTopologies(t *testing.T) {
	ctx := context.Background()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	//nolint:exhaustruct
	w := &Workflow{db: database}

	t.Run("two deployments that together exceed the quota cannot both reserve", func(t *testing.T) {
		workspaceID := workspaceWithMemoryQuota(t, ctx, seeder, 256)
		requests := []reserveTopologiesRequest{
			reservationFor(ctx, seeder, workspaceID),
			reservationFor(ctx, seeder, workspaceID),
		}

		messages := make([]string, len(requests))
		errs := make([]error, len(requests))
		var wg sync.WaitGroup
		for i, req := range requests {
			wg.Go(func() {
				res, err := w.reserveTopologies(ctx, req)
				messages[i], errs[i] = res.Message, err
			})
		}
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}
		require.ElementsMatch(t, []string{"", deployfail.MsgMemoryQuotaExceeded}, messages)

		reserved := 0
		for _, req := range requests {
			rows, err := database.FindDeploymentTopologyMinReplicas(ctx, req.Deployment.ID)
			require.NoError(t, err)
			reserved += len(rows)
		}
		require.Equal(t, 1, reserved, "exactly one deployment may hold topology rows")

		allocated, err := database.SumAllocatedResourcesByWorkspaceID(ctx, db.SumAllocatedResourcesByWorkspaceIDParams{
			WorkspaceID:         workspaceID,
			ExcludeDeploymentID: uid.New(uid.DeploymentPrefix),
		})
		require.NoError(t, err)
		require.EqualValues(t, 256, allocated.TotalMemoryMib)
	})

	t.Run("a re-run for the same deployment does not count its own rows", func(t *testing.T) {
		workspaceID := workspaceWithMemoryQuota(t, ctx, seeder, 256)
		req := reservationFor(ctx, seeder, workspaceID)
		for range 2 {
			res, err := w.reserveTopologies(ctx, req)
			require.NoError(t, err)
			require.Empty(t, res.Message)
		}
	})

	t.Run("deletion prevents topology recreation", func(t *testing.T) {
		workspaceID := workspaceWithMemoryQuota(t, ctx, seeder, 256)
		req := reservationFor(ctx, seeder, workspaceID)
		require.NoError(t, database.MarkEnvironmentDeleting(ctx, db.MarkEnvironmentDeletingParams{
			ID: req.Deployment.EnvironmentID, DeletingAt: sql.NullInt64{Int64: 1, Valid: true},
		}))
		_, err := w.reserveTopologies(ctx, req)
		require.Error(t, err)
		rows, err := database.FindDeploymentTopologyMinReplicas(ctx, req.Deployment.ID)
		require.NoError(t, err)
		require.Empty(t, rows)
	})
}

// The limits upsert writes every column, so the values other than memory
// mirror the seeder's
func workspaceWithMemoryQuota(t *testing.T, ctx context.Context, seeder *seed.Seeder, memoryMib uint32) string {
	t.Helper()
	ws := seeder.CreateWorkspace(ctx)
	require.NoError(t, seeder.DB.UpsertLimit(ctx, db.UpsertLimitParams{
		WorkspaceID:                           ws.ID,
		ApiBillableOperationsCountMaxPerMonth: 1_000_000,
		ApiRequestsCountMaxPerMinute:          sql.NullInt32{Valid: false, Int32: 0},
		LogsRetentionDaysMax:                  30,
		LogsAuditRetentionDaysMax:             30,
		TeamEnabled:                           false,
		CpuCoresMax:                           10,
		CpuCoresMaxPerInstance:                2,
		MemoryMibMax:                          memoryMib,
		MemoryMibMaxPerInstance:               4_096,
		StorageMibMax:                         51_200,
		StorageMibMaxPerInstance:              10_240,
		BuildsConcurrentMax:                   1,
		CustomDomainsMax:                      0,
		AutoscalingReplicasMax:                0,
	}))
	return ws.ID
}

func reservationFor(ctx context.Context, seeder *seed.Seeder, workspaceID string) reserveTopologiesRequest {
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "Reservation", Slug: uid.DNS1035(),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "Reservation", Slug: uid.DNS1035(),
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID,
		Slug: "production", Kind: mysqltype.EnvironmentKindProduction,
	})
	created := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusBuilding,
	})
	return reserveTopologiesRequest{
		Deployment: db.FindDeploymentForDeployRow{
			ID:            created.ID,
			WorkspaceID:   created.WorkspaceID,
			EnvironmentID: created.EnvironmentID,
			CpuMillicores: created.CpuMillicores,
			MemoryMib:     created.MemoryMib,
			StorageMib:    created.StorageMib,
		},
		Topologies: []db.InsertDeploymentTopologyParams{{
			WorkspaceID:            created.WorkspaceID,
			DeploymentID:           created.ID,
			RegionID:               uid.New(uid.RegionPrefix),
			AutoscalingReplicasMin: 1,
			AutoscalingReplicasMax: 1,
			DesiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
		}},
	}
}
