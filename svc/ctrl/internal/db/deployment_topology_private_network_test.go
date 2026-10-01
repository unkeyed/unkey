package db_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeploymentTopologyPrivateNetworkFollowsStoredDecision(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	workspace := seeder.CreateWorkspace(ctx)
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Project",
		Slug:        "project",
	})
	api := seeder.CreateApp(ctx, seed.CreateAppRequest{ID: uid.New(uid.AppPrefix), WorkspaceID: workspace.ID, ProjectID: project.ID, Name: "API", Slug: "api"})
	target := seeder.CreateApp(ctx, seed.CreateAppRequest{ID: uid.New(uid.AppPrefix), WorkspaceID: workspace.ID, ProjectID: project.ID, Name: "DB", Slug: "db"})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       api.ID,
		Slug:        "production",
		Kind:        dbtype.EnvironmentKindProduction,
	})
	region := seeder.CreateRegion(ctx, seed.CreateRegionRequest{Name: uid.New("region"), Platform: "kubernetes"})

	want := map[string]bool{}
	for _, enabled := range []bool{true, false} {
		deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			AppID:         api.ID,
			EnvironmentID: environment.ID,
			Status:        dbtype.DeploymentsStatusReady,
			Capabilities:  dbtype.DeploymentCapabilities{PrivateNetworking: enabled},
		})
		require.NoError(t, database.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
			WorkspaceID:            workspace.ID,
			DeploymentID:           deployment.ID,
			RegionID:               region.ID,
			AutoscalingReplicasMin: 1,
			AutoscalingReplicasMax: 1,
			DesiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
			CreatedAt:              time.Now().UnixMilli(),
		}))
		want[deployment.ID] = enabled
	}

	enrolled := func() map[string]bool {
		t.Helper()
		rows, listErr := database.ListAllDeploymentTopologiesByRegion(ctx, db.ListAllDeploymentTopologiesByRegionParams{RegionID: region.ID, AfterPk: 0, Limit: 10})
		require.NoError(t, listErr)
		byDeployment := map[string]bool{}
		for _, row := range rows {
			byDeployment[row.DeploymentID] = row.DeploymentCapabilities.PrivateNetworking
			found, findErr := database.FindDeploymentTopologyByDeploymentAndRegion(ctx, db.FindDeploymentTopologyByDeploymentAndRegionParams{DeploymentID: row.DeploymentID, RegionID: region.ID})
			require.NoError(t, findErr)
			require.Equal(t, row.DeploymentCapabilities, found.Capabilities, "both topology reads agree for %s", row.DeploymentID)
		}
		return byDeployment
	}

	require.Equal(t, want, enrolled(), "without connections")

	seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
		WorkspaceID:         workspace.ID,
		ProjectID:           project.ID,
		CallerAppID:         api.ID,
		CallerEnvironmentID: environment.ID,
		TargetAppID:         target.ID,
		Name:                "database",
	})
	require.Equal(t, want, enrolled(), "adding a connection changes no running deployment")

	require.NoError(t, database.DeleteAppById(ctx, target.ID))
	require.Equal(t, want, enrolled(), "deleting every connection changes no running deployment")
}
