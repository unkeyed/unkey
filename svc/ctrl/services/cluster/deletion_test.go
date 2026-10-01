package cluster

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type deletionFixture struct {
	database      db.Database
	service       *Service
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	deployment    db.Deployment
	regions       []db.Region
	clusterKey    *ctrlv1.ClusterKey
}

func newDeletionFixture(t *testing.T, regionCount int) deletionFixture {
	t.Helper()
	ctx := t.Context()
	mysqlConfig := containers.MySQL(t)
	database, err := db.New(mysqlConfig.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspaceID, Name: "deletion", Slug: uid.DNS1035(),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, Name: "deletion", Slug: uid.DNS1035(),
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID,
		Slug: "production", Kind: mysqltype.EnvironmentKindProduction,
	})
	deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID, EnvironmentID: environment.ID,
		Status: mysqltype.DeploymentsStatusReady,
	})

	regions := make([]db.Region, 0, regionCount)
	for i := 0; i < regionCount; i++ {
		region := seeder.CreateRegion(ctx, seed.CreateRegionRequest{Name: uid.DNS1035(), Platform: "deletion-test"})
		require.NoError(t, database.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
			WorkspaceID: workspaceID, DeploymentID: deployment.ID, RegionID: region.ID,
			AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
			DesiredStatus: db.DeploymentTopologyDesiredStatusRunning, CreatedAt: time.Now().UnixMilli(),
		}))
		regions = append(regions, region)
	}

	clusterKey := &ctrlv1.ClusterKey{CellId: uid.New("cell"), Platform: regions[0].Platform, Region: regions[0].Name}
	require.NoError(t, database.UpsertCluster(ctx, db.UpsertClusterParams{
		ID: uid.New("cluster"), CellID: sql.NullString{String: clusterKey.CellId, Valid: true},
		RegionID: regions[0].ID, LastHeartbeatAt: uint64(time.Now().UnixMilli()),
	}))
	clusterCache, err := cache.New(cache.Config[clusterCacheKey, db.FindClusterRow]{
		Fresh: time.Minute, Stale: time.Minute, MaxSize: 4, Resource: "deletion_test", Clock: clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(clusterCache.Close)

	return deletionFixture{
		database: database, service: &Service{db: database, bearer: "test-token", clusterCache: clusterCache},
		workspaceID: workspaceID, projectID: project.ID, appID: app.ID, environmentID: environment.ID,
		deployment: deployment, regions: regions, clusterKey: clusterKey,
	}
}

func TestDeletionDesiredStateForDeletingAndMissingParents(t *testing.T) {
	for _, test := range []struct {
		name         string
		removeParent func(*testing.T, context.Context, deletionFixture)
	}{
		{name: "deleting project", removeParent: func(t *testing.T, ctx context.Context, f deletionFixture) {
			t.Helper()
			require.NoError(t, f.database.MarkProjectDeleting(ctx, db.MarkProjectDeletingParams{ID: f.projectID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}}))
		}},
		{name: "deleting app", removeParent: func(t *testing.T, ctx context.Context, f deletionFixture) {
			t.Helper()
			require.NoError(t, f.database.MarkAppDeleting(ctx, db.MarkAppDeletingParams{ID: f.appID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}}))
		}},
		{name: "deleting environment", removeParent: func(t *testing.T, ctx context.Context, f deletionFixture) {
			t.Helper()
			require.NoError(t, f.database.MarkEnvironmentDeleting(ctx, db.MarkEnvironmentDeletingParams{ID: f.environmentID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}}))
		}},
		{name: "missing project", removeParent: func(t *testing.T, ctx context.Context, f deletionFixture) {
			t.Helper()
			require.NoError(t, f.database.DeleteProjectById(ctx, f.projectID))
		}},
		{name: "missing app", removeParent: func(t *testing.T, ctx context.Context, f deletionFixture) {
			t.Helper()
			require.NoError(t, f.database.DeleteAppById(ctx, f.appID))
		}},
		{name: "missing environment", removeParent: func(t *testing.T, ctx context.Context, f deletionFixture) {
			t.Helper()
			require.NoError(t, f.database.DeleteEnvironmentById(ctx, f.environmentID))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newDeletionFixture(t, 1)
			test.removeParent(t, t.Context(), f)
			require.NoError(t, f.database.StopDeploymentTopologiesByEnvironment(t.Context(), db.StopDeploymentTopologiesByEnvironmentParams{
				EnvironmentID: f.environmentID, UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			}))

			point, err := f.database.FindDeploymentTopologyByDeploymentAndRegion(t.Context(), db.FindDeploymentTopologyByDeploymentAndRegionParams{
				DeploymentID: f.deployment.ID, RegionID: f.regions[0].ID,
			})
			require.NoError(t, err)
			pointState, err := deploymentRowToState(point)
			require.NoError(t, err)
			require.True(t, pointState.GetDelete().GetPermanent())

			rows, err := f.database.ListAllDeploymentTopologiesByRegion(t.Context(), db.ListAllDeploymentTopologiesByRegionParams{
				RegionID: f.regions[0].ID, Limit: 10,
			})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			fullState, err := deploymentRowToState(rows[0])
			require.NoError(t, err)
			require.True(t, fullState.GetDelete().GetPermanent())
		})
	}
}

func TestDeletionDesiredStateKeepsActiveDeployment(t *testing.T) {
	f := newDeletionFixture(t, 1)
	point, err := f.database.FindDeploymentTopologyByDeploymentAndRegion(t.Context(), db.FindDeploymentTopologyByDeploymentAndRegionParams{
		DeploymentID: f.deployment.ID, RegionID: f.regions[0].ID,
	})
	require.NoError(t, err)
	state, err := deploymentRowToState(point)
	require.NoError(t, err)
	require.NotNil(t, state.GetApply())

	rows, err := f.database.ListAllDeploymentTopologiesByRegion(t.Context(), db.ListAllDeploymentTopologiesByRegionParams{RegionID: f.regions[0].ID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	state, err = deploymentRowToState(rows[0])
	require.NoError(t, err)
	require.NotNil(t, state.GetApply())
}

func TestDeletionReportsRequireExplicitConfirmation(t *testing.T) {
	f := newDeletionFixture(t, 1)
	f.createInstance(t, f.regions[0])
	require.NoError(t, f.database.MarkEnvironmentDeleting(t.Context(), db.MarkEnvironmentDeletingParams{
		ID: f.environmentID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	}))

	req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
		Cluster: f.clusterKey,
		Change:  &ctrlv1.ReportDeploymentStatusRequest_Delete_{Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{K8SName: f.deployment.K8sName}},
	})
	req.Header().Set("Authorization", "Bearer test-token")
	_, err := f.service.ReportDeploymentStatus(t.Context(), req)
	require.NoError(t, err)
	f.requireTopologyCount(t, 1)
	f.requireInstanceCount(t, f.regions[0], 0)
}

func TestStoppedDeploymentFullSyncRecoversLostStatusReport(t *testing.T) {
	f := newDeletionFixture(t, 1)
	f.createInstance(t, f.regions[0])
	now := sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}
	require.NoError(t, f.database.StopDeploymentsByEnvironment(t.Context(), db.StopDeploymentsByEnvironmentParams{EnvironmentID: f.environmentID, UpdatedAt: now}))
	require.NoError(t, f.database.StopDeploymentTopologiesByEnvironment(t.Context(), db.StopDeploymentTopologiesByEnvironmentParams{EnvironmentID: f.environmentID, UpdatedAt: now}))

	for attempt := 0; attempt < 2; attempt++ {
		rows, err := f.database.ListAllDeploymentTopologiesByRegion(t.Context(), db.ListAllDeploymentTopologiesByRegionParams{RegionID: f.regions[0].ID, Limit: 10})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		state, err := deploymentRowToState(rows[0])
		require.NoError(t, err)
		require.NotNil(t, state.GetDelete())
		require.False(t, state.GetDelete().GetPermanent())
		if attempt == 0 {
			f.requireInstanceCount(t, f.regions[0], 1)
			continue
		}
		req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
			Cluster: f.clusterKey,
			Change: &ctrlv1.ReportDeploymentStatusRequest_Delete_{Delete: &ctrlv1.ReportDeploymentStatusRequest_Delete{
				K8SName: state.GetDelete().GetK8SName(), DeploymentId: f.deployment.ID,
			}},
		})
		req.Header().Set("Authorization", "Bearer test-token")
		_, err = f.service.ReportDeploymentStatus(t.Context(), req)
		require.NoError(t, err)
	}
	f.requireInstanceCount(t, f.regions[0], 0)
	f.requireTopologyCount(t, 1)
	deployment, err := f.database.FindDeploymentById(t.Context(), f.deployment.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusStopped, deployment.Status)
}

func TestDeletionConfirmationIsRegionalIdempotentAndValidated(t *testing.T) {
	f := newDeletionFixture(t, 2)
	for _, region := range f.regions {
		f.createInstance(t, region)
	}
	report := &ctrlv1.ReportDeploymentStatusRequest_Delete{
		DeploymentId: f.deployment.ID, K8SName: f.deployment.K8sName, RemovalConfirmed: true,
	}
	queries := db.NewQueries(f.database.RW())

	err := confirmDeploymentRemoval(t.Context(), queries, f.regions[0].ID, report)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	f.requireTopologyCount(t, 2)

	require.NoError(t, f.database.MarkEnvironmentDeleting(t.Context(), db.MarkEnvironmentDeletingParams{
		ID: f.environmentID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	}))
	err = confirmDeploymentRemoval(t.Context(), queries, f.regions[0].ID, &ctrlv1.ReportDeploymentStatusRequest_Delete{
		DeploymentId: f.deployment.ID, K8SName: "wrong", RemovalConfirmed: true,
	})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	f.requireTopologyCount(t, 2)

	require.NoError(t, confirmDeploymentRemoval(t.Context(), queries, f.regions[0].ID, report))
	require.NoError(t, confirmDeploymentRemoval(t.Context(), queries, f.regions[0].ID, report))
	f.requireTopologyCount(t, 1)
	f.requireInstanceCount(t, f.regions[0], 0)
	f.requireInstanceCount(t, f.regions[1], 1)
}

func TestOwnerlessRemovalConfirmationAcceptsDeletingEnvironment(t *testing.T) {
	f := newDeletionFixture(t, 1)
	f.createInstance(t, f.regions[0])
	report := &ctrlv1.ReportDeploymentStatusRequest_Delete{DeploymentId: f.deployment.ID, RemovalConfirmed: true}
	queries := db.NewQueries(f.database.RW())
	err := confirmDeploymentRemoval(t.Context(), queries, f.regions[0].ID, report)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	f.requireTopologyCount(t, 1)
	f.requireInstanceCount(t, f.regions[0], 1)

	require.NoError(t, f.database.MarkEnvironmentDeleting(t.Context(), db.MarkEnvironmentDeletingParams{
		ID: f.environmentID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	}))
	require.NoError(t, confirmDeploymentRemoval(t.Context(), queries, f.regions[0].ID, report))
	f.requireTopologyCount(t, 0)
	f.requireInstanceCount(t, f.regions[0], 0)
	_, err = f.database.FindDeploymentById(t.Context(), f.deployment.ID)
	require.NoError(t, err)
}

func TestLateInstanceUpdateDoesNotRecreateInstancesAfterDeletion(t *testing.T) {
	f := newDeletionFixture(t, 1)
	require.NoError(t, f.database.MarkAppDeleting(t.Context(), db.MarkAppDeletingParams{
		ID: f.appID, DeletingAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	}))
	req := connect.NewRequest(&ctrlv1.ReportDeploymentStatusRequest{
		Cluster: f.clusterKey,
		Change: &ctrlv1.ReportDeploymentStatusRequest_Update_{Update: &ctrlv1.ReportDeploymentStatusRequest_Update{
			K8SName:   f.deployment.K8sName,
			Instances: []*ctrlv1.ReportDeploymentStatusRequest_Update_Instance{{K8SName: "late-pod", Address: "10.0.0.1"}},
		}},
	})
	req.Header().Set("Authorization", "Bearer test-token")
	_, err := f.service.ReportDeploymentStatus(t.Context(), req)
	require.NoError(t, err)
	f.requireInstanceCount(t, f.regions[0], 0)
}

func TestLockActiveEnvironmentRejectsDeletingAncestor(t *testing.T) {
	for _, ancestor := range []string{"project", "app", "environment"} {
		t.Run(ancestor, func(t *testing.T) {
			f := newDeletionFixture(t, 1)
			now := sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}
			switch ancestor {
			case "project":
				require.NoError(t, f.database.MarkProjectDeleting(t.Context(), db.MarkProjectDeletingParams{ID: f.projectID, DeletingAt: now}))
			case "app":
				require.NoError(t, f.database.MarkAppDeleting(t.Context(), db.MarkAppDeletingParams{ID: f.appID, DeletingAt: now}))
			case "environment":
				require.NoError(t, f.database.MarkEnvironmentDeleting(t.Context(), db.MarkEnvironmentDeletingParams{ID: f.environmentID, DeletingAt: now}))
			}
			_, err := f.database.LockActiveEnvironment(t.Context(), f.environmentID)
			require.True(t, db.IsNotFound(err))
		})
	}
}

func (f deletionFixture) createInstance(t *testing.T, region db.Region) {
	t.Helper()
	require.NoError(t, f.database.UpsertInstance(t.Context(), db.UpsertInstanceParams{
		ID: uid.New(uid.InstancePrefix), DeploymentID: f.deployment.ID, WorkspaceID: f.workspaceID,
		ProjectID: f.projectID, AppID: f.appID, RegionID: region.ID, K8sName: uid.DNS1035(),
		Address: "10.0.0.1", CpuMillicores: 100, MemoryMib: 128, Status: db.InstancesStatusRunning,
	}))
}

func (f deletionFixture) requireTopologyCount(t *testing.T, want int) {
	t.Helper()
	var got int
	require.NoError(t, f.database.RW().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM deployment_topology WHERE deployment_id = ?", f.deployment.ID).Scan(&got))
	require.Equal(t, want, got)
}

func TestActiveEnvironmentLockSerializesProjectDeletion(t *testing.T) {
	f := newDeletionFixture(t, 1)
	tx, err := f.database.RW().Begin(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() {
		err := tx.Rollback()
		if err != sql.ErrTxDone {
			require.NoError(t, err)
		}
	})
	_, err = db.NewQueries(tx).LockActiveEnvironment(t.Context(), f.environmentID)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()
	mark := db.MarkProjectDeletingParams{ID: f.projectID, DeletingAt: sql.NullInt64{Int64: 1, Valid: true}}
	err = f.database.MarkProjectDeleting(ctx, mark)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, tx.Commit())

	require.NoError(t, f.database.MarkProjectDeleting(t.Context(), mark))
	_, err = f.database.LockActiveEnvironment(t.Context(), f.environmentID)
	require.True(t, db.IsNotFound(err))
}

func (f deletionFixture) requireInstanceCount(t *testing.T, region db.Region, want int) {
	t.Helper()
	var got int
	require.NoError(t, f.database.RW().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM instances WHERE deployment_id = ? AND region_id = ?", f.deployment.ID, region.ID).Scan(&got))
	require.Equal(t, want, got)
}
