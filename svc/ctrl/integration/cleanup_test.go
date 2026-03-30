//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	workerapp "github.com/unkeyed/unkey/svc/ctrl/worker/app"
	workerenvironment "github.com/unkeyed/unkey/svc/ctrl/worker/environment"
	workerproject "github.com/unkeyed/unkey/svc/ctrl/worker/project"

	restatetest "github.com/restatedev/sdk-go/testing"
)

// TestProjectDeletion_CleansUpAllData verifies the full project → app →
// environment deletion cascade by running the actual Restate virtual objects
// against a real Restate server (via testcontainers).
//
// It seeds a project with a full resource tree, then calls ProjectService/Delete
// through the Restate ingress and asserts that every table is cleaned up.
//
// Note: instances are NOT covered here. They are cleaned up asynchronously by
// the reconciler once k8s pods are removed, not by the project delete handler.
func TestProjectDeletion_CleansUpAllData(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	// Start Restate with all three deletion VOs bound.
	tEnv := restatetest.Start(t,
		hydrav1.NewProjectServiceServer(workerproject.New(workerproject.Config{DB: h.DB})),
		hydrav1.NewAppServiceServer(workerapp.New(workerapp.Config{DB: h.DB})),
		hydrav1.NewEnvironmentServiceServer(workerenvironment.New(workerenvironment.Config{DB: h.DB})),
	)

	workspaceID := h.Seed.Resources.UserWorkspace.ID
	now := time.Now().UnixMilli()

	// --- Seed a full project resource tree ---

	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: workspaceID,
		Name:        "cleanup-test-project",
		Slug:        uid.New("slug"),
	})

	app := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "cleanup-test-app",
		Slug:          "default",
		DefaultBranch: "main",
	})

	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:             uid.New("env"),
		WorkspaceID:    workspaceID,
		ProjectID:      project.ID,
		AppID:          app.ID,
		Slug:           "production",
		Description:    "",
		SentinelConfig: []byte("{}"),
	})
	// CreateEnvironment already inserts build and runtime settings.

	deployment := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
	})

	// Region (needed for topology, sentinels, cilium policies)
	regionID := uid.New(uid.RegionPrefix)
	err := db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
		ID:       regionID,
		Name:     "test-cleanup",
		Platform: "test",
	})
	require.NoError(t, err)

	region, err := db.Query.FindRegionByPlatformAndName(ctx, h.DB.RO(), db.FindRegionByPlatformAndNameParams{
		Name:     "test-cleanup",
		Platform: "test",
	})
	require.NoError(t, err)

	// Deployment topology
	err = db.Query.InsertDeploymentTopology(ctx, h.DB.RW(), db.InsertDeploymentTopologyParams{
		WorkspaceID:                workspaceID,
		DeploymentID:               deployment.ID,
		RegionID:                   region.ID,
		DesiredReplicas:            1,
		AutoscalingReplicasMin:     1,
		AutoscalingReplicasMax:     1,
		AutoscalingThresholdCpu:    sql.NullInt16{Valid: false},
		AutoscalingThresholdMemory: sql.NullInt16{Valid: false},
		DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
		Version:                    1,
		CreatedAt:                  now,
	})
	require.NoError(t, err)

	// Sentinel
	err = db.Query.InsertSentinel(ctx, h.DB.RW(), db.InsertSentinelParams{
		ID:                uid.New("sen"),
		WorkspaceID:       workspaceID,
		EnvironmentID:     env.ID,
		ProjectID:         project.ID,
		K8sAddress:        "http://localhost:9090",
		K8sName:           uid.New("k8s"),
		RegionID:          region.ID,
		Image:             "sentinel:1.0",
		Health:            db.SentinelsHealthHealthy,
		DesiredReplicas:   1,
		AvailableReplicas: 1,
		CpuMillicores:     100,
		MemoryMib:         128,
		Version:           1,
		CreatedAt:         now,
	})
	require.NoError(t, err)

	// Cilium network policy
	err = db.Query.InsertCiliumNetworkPolicy(ctx, h.DB.RW(), db.InsertCiliumNetworkPolicyParams{
		ID:            uid.New("cnp"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		DeploymentID:  deployment.ID,
		K8sName:       uid.New("k8s"),
		K8sNamespace:  "test-ns",
		RegionID:      region.ID,
		Policy:        json.RawMessage(`{"apiVersion":"cilium.io/v2"}`),
		Version:       1,
		CreatedAt:     now,
	})
	require.NoError(t, err)

	// Frontline route
	err = db.Query.InsertFrontlineRoute(ctx, h.DB.RW(), db.InsertFrontlineRouteParams{
		ID:                       uid.New("fr"),
		ProjectID:                project.ID,
		AppID:                    app.ID,
		DeploymentID:             deployment.ID,
		EnvironmentID:            env.ID,
		FullyQualifiedDomainName: "cleanup-test.example.com",
		Sticky:                   db.FrontlineRoutesStickyNone,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Valid: false},
	})
	require.NoError(t, err)

	// GitHub repo connection
	err = db.Query.InsertGithubRepoConnection(ctx, h.DB.RW(), db.InsertGithubRepoConnectionParams{
		ProjectID:          project.ID,
		AppID:              app.ID,
		InstallationID:     12345,
		RepositoryID:       67890,
		RepositoryFullName: "unkeyed/test-repo",
		CreatedAt:          now,
		UpdatedAt:          sql.NullInt64{Valid: false},
	})
	require.NoError(t, err)

	// Deployment step
	err = db.Query.InsertDeploymentStep(ctx, h.DB.RW(), db.InsertDeploymentStepParams{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		DeploymentID:  deployment.ID,
		Step:          db.DeploymentStepsStepBuilding,
		StartedAt:     uint64(now),
	})
	require.NoError(t, err)

	// App regional settings
	err = db.Query.UpsertAppRegionalSettings(ctx, h.DB.RW(), db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		RegionID:      region.ID,
		Replicas:      2,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Valid: false},
	})
	require.NoError(t, err)

	// App environment variable
	err = db.Query.InsertAppEnvironmentVariable(ctx, h.DB.RW(), db.InsertAppEnvironmentVariableParams{
		ID:            uid.New("aev"),
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		EnvKey:        "TEST_KEY",
		Value:         "test_value",
		CreatedAt:     now,
	})
	require.NoError(t, err)

	// Queries to verify each table has exactly one row before deletion
	// and zero rows after deletion.
	checks := []struct {
		query string
		arg   string
	}{
		{"SELECT COUNT(*) FROM projects WHERE id = ?", project.ID},
		{"SELECT COUNT(*) FROM apps WHERE id = ?", app.ID},
		{"SELECT COUNT(*) FROM environments WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM deployments WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM deployment_topology WHERE deployment_id = ?", deployment.ID},
		{"SELECT COUNT(*) FROM sentinels WHERE project_id = ?", project.ID},
		{"SELECT COUNT(*) FROM cilium_network_policies WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM frontline_routes WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM github_repo_connections WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM deployment_steps WHERE deployment_id = ?", deployment.ID},
		{"SELECT COUNT(*) FROM app_build_settings WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM app_runtime_settings WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM app_regional_settings WHERE app_id = ?", app.ID},
		{"SELECT COUNT(*) FROM app_environment_variables WHERE app_id = ?", app.ID},
	}

	// --- Verify all rows exist before deletion ---

	for _, c := range checks {
		require.Equal(t, 1, countRows(t, ctx, h.DB, c.query, c.arg))
	}

	// --- Trigger deletion via Restate ingress ---

	projectClient := hydrav1.NewProjectServiceIngressClient(tEnv.Ingress(), project.ID)
	_, err = projectClient.Delete().Request(ctx, &hydrav1.DeleteProjectRequest{})
	require.NoError(t, err)

	// The project handler fires off app deletions via .Send() (durable but async).
	// The app handler fires off environment deletions via .Send() (also async).
	// Poll each table until it's empty.
	for _, c := range checks {
		c := c
		require.Eventually(t, func() bool {
			return countRows(t, ctx, h.DB, c.query, c.arg) == 0
		}, 30*time.Second, 250*time.Millisecond, "timed out waiting for: %s", c.query)
	}
}

// countRows executes a query that returns a single COUNT(*) value.
//
//nolint:gosec // queries are test constants, not user input
func countRows(t *testing.T, ctx context.Context, database db.Database, query string, args ...any) int {
	t.Helper()
	var count int
	err := database.RO().QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err)
	return count
}
