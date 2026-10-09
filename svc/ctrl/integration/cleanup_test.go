//go:build integration

package integration

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	workerapp "github.com/unkeyed/unkey/svc/ctrl/worker/app"
	workerenvironment "github.com/unkeyed/unkey/svc/ctrl/worker/environment"
	workerproject "github.com/unkeyed/unkey/svc/ctrl/worker/project"

	restatetest "github.com/restatedev/sdk-go/testing"
)

// TestProjectDeletion_CleansUpAllData verifies the full project → app →
// environment deletion cascade by running the actual Restate virtual objects
// against a real Restate server.
//
// It seeds a project with a full resource tree, then calls ProjectService/Delete
// through the Restate ingress and asserts that every table is cleaned up.
//
// Note: instances are NOT covered here. They are cleaned up asynchronously by
// the reconciler once k8s pods are removed, not by the project delete handler.
func TestProjectDeletion_CleansUpAllData(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	auditlogsService, err := auditlogs.New(auditlogs.Config{DB: h.DB})
	require.NoError(t, err)

	// The environment delete handler only calls Admin to cancel a deployment's
	// in-flight Restate invocation, and seeded deployments have no invocation id,
	// so Admin is never exercised here. It just has to be non-nil.
	envSvc, err := workerenvironment.New(workerenvironment.Config{
		DB:        h.DB,
		Admin:     restateadmin.New(restateadmin.Config{BaseURL: "http://127.0.0.1:9070", APIKey: ""}),
		Auditlogs: auditlogsService,
	})
	require.NoError(t, err)

	projSvc, err := workerproject.New(workerproject.Config{DB: h.DB, Auditlogs: auditlogsService})
	require.NoError(t, err)
	appSvc, err := workerapp.New(workerapp.Config{DB: h.DB, Auditlogs: auditlogsService})
	require.NoError(t, err)

	// Start Restate with all three deletion VOs bound.
	tEnv := restatetest.Start(t,
		hydrav1.NewProjectServiceServer(projSvc),
		hydrav1.NewAppServiceServer(appSvc),
		hydrav1.NewEnvironmentServiceServer(envSvc),
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
		ID:          uid.New("app"),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "cleanup-test-app",
		Slug:        "default",
	})

	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:             uid.New("env"),
		WorkspaceID:    workspaceID,
		ProjectID:      project.ID,
		AppID:          app.ID,
		Slug:           "production",
		Kind:           mysqltype.EnvironmentKindProduction,
		Description:    "",
		SentinelConfig: []byte("{}"),
	})
	// CreateEnvironment already inserts build and runtime settings.

	deployment := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	// Region (needed for topology and cilium policies)
	regionID := uid.New(uid.RegionPrefix)
	err = h.DB.UpsertRegion(ctx, db.UpsertRegionParams{
		ID:       regionID,
		Name:     "test-cleanup",
		Platform: "test",
	})
	require.NoError(t, err)

	region, err := h.DB.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Name:     "test-cleanup",
		Platform: "test",
	})
	require.NoError(t, err)

	// Deployment topology
	err = h.DB.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
		WorkspaceID:                workspaceID,
		DeploymentID:               deployment.ID,
		RegionID:                   region.ID,
		AutoscalingReplicasMin:     1,
		AutoscalingReplicasMax:     1,
		AutoscalingThresholdCpu:    sql.NullInt16{Valid: false},
		AutoscalingThresholdMemory: sql.NullInt16{Valid: false},
		DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
		CreatedAt:                  now,
	})
	require.NoError(t, err)

	// Cilium network policy
	policy, err := json.Marshal(struct {
		APIVersion string `json:"apiVersion"`
	}{APIVersion: "cilium.io/v2"})
	require.NoError(t, err)
	err = h.DB.InsertCiliumNetworkPolicy(ctx, db.InsertCiliumNetworkPolicyParams{
		ID:            uid.New("cnp"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		DeploymentID:  deployment.ID,
		K8sName:       uid.New("k8s"),
		K8sNamespace:  "test-ns",
		RegionID:      region.ID,
		Policy:        policy,
		CreatedAt:     now,
	})
	require.NoError(t, err)

	// Frontline route
	err = h.DB.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
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
	err = h.DB.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspaceID,
		ProjectID:          project.ID,
		AppID:              app.ID,
		InstallationID:     12345,
		RepositoryID:       67890,
		RepositoryFullName: "unkeyed/test-repo",
		DefaultBranch:      sql.NullString{Valid: false},
		CreatedAt:          now,
		UpdatedAt:          sql.NullInt64{Valid: false},
	})
	require.NoError(t, err)

	err = h.DB.InsertAppSourceOci(ctx, db.InsertAppSourceOciParams{
		WorkspaceID:    workspaceID,
		AppID:          app.ID,
		ImageReference: "index.docker.io/library/nginx:1.27",
		CreatedAt:      now,
		UpdatedAt:      sql.NullInt64{Valid: false},
	})
	require.NoError(t, err)

	// Deployment step
	err = h.DB.InsertDeploymentStep(ctx, db.InsertDeploymentStepParams{
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
	err = h.DB.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
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
	err = h.DB.InsertAppEnvironmentVariable(ctx, db.InsertAppEnvironmentVariableParams{
		ID:            uid.New("aev"),
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		EnvKey:        "TEST_KEY",
		Value:         "test_value",
		CreatedAt:     now,
	})
	require.NoError(t, err)

	// Each table has exactly one row before deletion and zero rows after.
	checks := []struct {
		table string
		count func() (int64, error)
	}{
		{"projects", func() (int64, error) { return h.DB.CountProjectsById(ctx, project.ID) }},
		{"apps", func() (int64, error) { return h.DB.CountAppsById(ctx, app.ID) }},
		{"environments", func() (int64, error) { return h.DB.CountEnvironmentsByAppId(ctx, app.ID) }},
		{"deployments", func() (int64, error) { return h.DB.CountDeploymentsByAppId(ctx, app.ID) }},
		{"deployment_topology", func() (int64, error) { return h.DB.CountDeploymentTopologiesByDeploymentId(ctx, deployment.ID) }},
		{"cilium_network_policies", func() (int64, error) { return h.DB.CountCiliumNetworkPoliciesByAppId(ctx, app.ID) }},
		{"frontline_routes", func() (int64, error) { return h.DB.CountFrontlineRoutesByAppId(ctx, app.ID) }},
		{"github_repo_connections", func() (int64, error) { return h.DB.CountGithubRepoConnectionsByAppId(ctx, app.ID) }},
		{"app_source_oci", func() (int64, error) { return h.DB.CountAppSourceOciByAppId(ctx, app.ID) }},
		{"deployment_steps", func() (int64, error) { return h.DB.CountDeploymentStepsByDeploymentId(ctx, deployment.ID) }},
		{"app_build_settings", func() (int64, error) { return h.DB.CountAppBuildSettingsByAppId(ctx, app.ID) }},
		{"app_runtime_settings", func() (int64, error) { return h.DB.CountAppRuntimeSettingsByAppId(ctx, app.ID) }},
		{"app_regional_settings", func() (int64, error) { return h.DB.CountAppRegionalSettingsByAppId(ctx, app.ID) }},
		{"app_environment_variables", func() (int64, error) { return h.DB.CountAppEnvironmentVariablesByAppId(ctx, app.ID) }},
	}

	// --- Verify all rows exist before deletion ---

	for _, c := range checks {
		n, err := c.count()
		require.NoError(t, err)
		require.Equal(t, int64(1), n, c.table)
	}

	// The API marks rows deleted before the worker runs.
	_, err = h.DB.RW().ExecContext(ctx, "UPDATE projects SET deleted_at_m = ? WHERE id = ?", now, project.ID)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(ctx, "UPDATE apps SET deleted_at_m = ? WHERE id = ?", now, app.ID)
	require.NoError(t, err)

	// --- Trigger deletion via Restate ingress ---

	projectClient := hydrav1.NewProjectServiceIngressClient(tEnv.Ingress(), project.ID)
	_, err = projectClient.Delete().Request(ctx, &hydrav1.DeleteProjectRequest{})
	require.NoError(t, err)

	// The project handler fires off app deletions via .Send() (durable but async).
	// The app handler fires off environment deletions via .Send() (also async).
	// Poll each table until it's empty.
	for _, c := range checks {
		require.Eventually(t, func() bool {
			n, err := c.count()
			require.NoError(t, err)
			return n == 0
		}, 30*time.Second, 250*time.Millisecond, "timed out waiting for %s to be empty", c.table)
	}
}
