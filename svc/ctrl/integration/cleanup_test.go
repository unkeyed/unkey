//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	workerapp "github.com/unkeyed/unkey/svc/ctrl/worker/app"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/resourcecleanup"
	workerenvironment "github.com/unkeyed/unkey/svc/ctrl/worker/environment"
	workerproject "github.com/unkeyed/unkey/svc/ctrl/worker/project"

	restate "github.com/restatedev/sdk-go"
	restatetest "github.com/restatedev/sdk-go/testing"
)

type resourceCleanupCron struct {
	hydrav1.UnimplementedCronServiceServer
	cleanup *resourcecleanup.Handler
}

func (c resourceCleanupCron) RunResourceCleanup(ctx restate.ObjectContext, req *hydrav1.RunResourceCleanupRequest) (*hydrav1.RunResourceCleanupResponse, error) {
	return c.cleanup.Handle(ctx, req)
}

func startDeletionRestate(t *testing.T, h *Harness) *restatetest.TestEnvironment {
	t.Helper()
	auditlogsService, err := auditlogs.New(auditlogs.Config{DB: h.DB})
	require.NoError(t, err)
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
	cleanup, err := resourcecleanup.New(resourcecleanup.Config{DB: h.DB, Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)

	return restatetest.Start(t,
		hydrav1.NewProjectServiceServer(projSvc),
		hydrav1.NewAppServiceServer(appSvc),
		hydrav1.NewEnvironmentServiceServer(envSvc),
		hydrav1.NewCronServiceServer(resourceCleanupCron{cleanup: cleanup}).
			ConfigureHandler("RunResourceCleanup", resourcecleanup.RetryPolicy()),
	)
}

func runResourceCleanup(t *testing.T, ctx context.Context, tEnv *restatetest.TestEnvironment) {
	t.Helper()
	_, err := hydrav1.NewCronServiceIngressClient(tEnv.Ingress(), "resource-cleanup").RunResourceCleanup().Request(ctx, &hydrav1.RunResourceCleanupRequest{})
	require.NoError(t, err)
}

func TestDeletionCleansUpOwnedData(t *testing.T) {
	for _, target := range []string{"environment", "app", "project"} {
		t.Run(target, func(t *testing.T) { testDeletionCleansUpOwnedData(t, target) })
	}
}

func testDeletionCleansUpOwnedData(t *testing.T, target string) {
	t.Helper()
	h := New(t)
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	tEnv := startDeletionRestate(t, h)

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
	err := h.DB.UpsertRegion(ctx, db.UpsertRegionParams{
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
		FullyQualifiedDomainName: uid.DNS1035() + ".example.com",
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

	domainID := uid.New(uid.DomainPrefix)
	hostname := uid.DNS1035() + ".example.com"
	policyID := uid.New("hap")
	portalID := uid.New(uid.PortalPrefix)
	deploymentSpecID := uid.New(uid.OpenApiSpecPrefix)
	portalSpecID := uid.New(uid.OpenApiSpecPrefix)
	sibling := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: uid.DNS1035(), DesiredState: mysqltype.DeploymentsDesiredStateRunning})
	siblingPortalID := uid.New(uid.PortalPrefix)
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{"UPDATE apps SET current_deployment_id = ? WHERE id = ?", []any{deployment.ID, app.ID}},
		{"UPDATE apps SET current_deployment_id = ? WHERE id = ?", []any{sibling.ID, sibling.AppID}},
		{`INSERT INTO custom_domains (id, workspace_id, project_id, app_id, environment_id, domain, challenge_type, verification_token, target_cname, created_at)
			VALUES (?, ?, ?, ?, ?, ?, 'HTTP-01', '', ?, 1)`, []any{domainID, workspaceID, project.ID, app.ID, env.ID, hostname, hostname}},
		{`INSERT INTO acme_challenges (workspace_id, domain_id, token, authorization, status, challenge_type, created_at, expires_at)
			VALUES (?, ?, '', '', 'waiting', 'HTTP-01', 1, ?)`, []any{workspaceID, domainID, now + time.Hour.Milliseconds()}},
		{`INSERT INTO certificates (id, workspace_id, hostname, certificate, encrypted_private_key, created_at)
			VALUES (?, ?, ?, 'certificate', 'encrypted', 1)`, []any{uid.New(uid.CertificatePrefix), workspaceID, hostname}},
		{`INSERT INTO horizontal_autoscaling_policies (id, workspace_id, replicas_min, replicas_max, created_at)
			VALUES (?, ?, 1, 3, 1)`, []any{policyID, workspaceID}},
		{"UPDATE app_regional_settings SET horizontal_autoscaling_policy_id = ? WHERE environment_id = ?", []any{policyID, env.ID}},
		{`INSERT INTO portals (id, workspace_id, project_id, app_id, slug, display_name, created_at)
			VALUES (?, ?, ?, ?, ?, 'test', 1)`, []any{portalID, workspaceID, project.ID, app.ID, portalID}},
		{`INSERT INTO portal_sessions (id, workspace_id, portal_id, external_id, scopes, exchange_code_hash, exchange_code_expires_at, created_at)
			VALUES (?, ?, ?, 'user', '{}', ?, ?, 1)`, []any{uid.New(uid.PortalSessionPrefix), workspaceID, portalID, portalID, now + time.Hour.Milliseconds()}},
		{`INSERT INTO portals (id, workspace_id, project_id, app_id, slug, display_name, created_at)
			VALUES (?, ?, ?, ?, ?, 'sibling', 1)`, []any{siblingPortalID, workspaceID, sibling.ProjectID, sibling.AppID, siblingPortalID}},
		{`INSERT INTO portal_sessions (id, workspace_id, portal_id, external_id, scopes, exchange_code_hash, exchange_code_expires_at, created_at)
			VALUES (?, ?, ?, 'user', '{}', ?, ?, 1)`, []any{uid.New(uid.PortalSessionPrefix), workspaceID, siblingPortalID, siblingPortalID, now + time.Hour.Milliseconds()}},
	} {
		_, err := h.DB.RW().ExecContext(ctx, statement.query, statement.args...)
		require.NoError(t, err)
	}
	for _, spec := range []db.UpsertOpenApiSpecParams{
		{ID: deploymentSpecID, WorkspaceID: workspaceID, DeploymentID: sql.NullString{Valid: true, String: deployment.ID}, Content: []byte("{}"), CreatedAt: now},
		{ID: portalSpecID, WorkspaceID: workspaceID, PortalID: sql.NullString{Valid: true, String: portalID}, Content: []byte("{}"), CreatedAt: now},
		{ID: uid.New(uid.OpenApiSpecPrefix), WorkspaceID: workspaceID, PortalID: sql.NullString{Valid: true, String: siblingPortalID}, Content: []byte("{}"), CreatedAt: now},
	} {
		require.NoError(t, h.DB.UpsertOpenApiSpec(ctx, spec))
	}
	appSurvives := target == "environment"
	checks := []struct {
		query    string
		arg      string
		survives bool
	}{
		{"SELECT COUNT(*) FROM projects WHERE id = ?", project.ID, target != "project"},
		{"SELECT COUNT(*) FROM apps WHERE id = ?", app.ID, appSurvives},
		{"SELECT COUNT(*) FROM environments WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM deployments WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM deployment_topology WHERE deployment_id = ?", deployment.ID, false},
		{"SELECT COUNT(*) FROM cilium_network_policies WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM frontline_routes WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM github_repo_connections WHERE app_id = ?", app.ID, appSurvives},
		{"SELECT COUNT(*) FROM app_source_oci WHERE app_id = ?", app.ID, appSurvives},
		{"SELECT COUNT(*) FROM deployment_steps WHERE deployment_id = ?", deployment.ID, false},
		{"SELECT COUNT(*) FROM app_build_settings WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM app_runtime_settings WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM app_regional_settings WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM app_environment_variables WHERE app_id = ?", app.ID, false},
		{"SELECT COUNT(*) FROM custom_domains WHERE id = ?", domainID, false},
		{"SELECT COUNT(*) FROM acme_challenges WHERE domain_id = ?", domainID, false},
		{"SELECT COUNT(*) FROM certificates WHERE hostname = ?", hostname, false},
		{"SELECT COUNT(*) FROM horizontal_autoscaling_policies WHERE id = ?", policyID, false},
		{"SELECT COUNT(*) FROM portals WHERE id = ?", portalID, appSurvives},
		{"SELECT COUNT(*) FROM portal_sessions WHERE portal_id = ?", portalID, appSurvives},
		{"SELECT COUNT(*) FROM openapi_specs WHERE id = ?", deploymentSpecID, false},
		{"SELECT COUNT(*) FROM openapi_specs WHERE id = ?", portalSpecID, appSurvives},
		{"SELECT COUNT(*) FROM projects WHERE id = ?", sibling.ProjectID, true},
		{"SELECT COUNT(*) FROM apps WHERE id = ?", sibling.AppID, true},
		{"SELECT COUNT(*) FROM environments WHERE id = ?", sibling.EnvironmentID, true},
		{"SELECT COUNT(*) FROM deployments WHERE id = ? AND desired_state = 'running'", sibling.ID, true},
		{"SELECT COUNT(*) FROM deployment_topology WHERE deployment_id = ? AND desired_status = 'running'", sibling.ID, true},
		{"SELECT COUNT(*) FROM portals WHERE id = ?", siblingPortalID, true},
		{"SELECT COUNT(*) FROM portal_sessions WHERE portal_id = ?", siblingPortalID, true},
		{"SELECT COUNT(*) FROM openapi_specs WHERE portal_id = ?", siblingPortalID, true},
	}

	// --- Verify all rows exist before deletion ---

	for _, c := range checks {
		require.Equal(t, 1, countRows(t, ctx, h.DB, c.query, c.arg))
	}

	secondRegion := h.Seed.CreateRegion(ctx, seed.CreateRegionRequest{Name: uid.DNS1035(), Platform: "test"})
	require.NoError(t, h.DB.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
		WorkspaceID: workspaceID, DeploymentID: deployment.ID, RegionID: secondRegion.ID,
		AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
		DesiredStatus: db.DeploymentTopologyDesiredStatusRunning, CreatedAt: now,
	}))
	h.Seed.CreateInstance(ctx, seed.CreateInstanceRequest{
		WorkspaceID: workspaceID, ProjectID: project.ID, AppID: app.ID,
		DeploymentID: deployment.ID, RegionID: region.ID, Address: "10.0.0.1",
	})
	deleteResource := func() error {
		switch target {
		case "environment":
			_, err := hydrav1.NewEnvironmentServiceIngressClient(tEnv.Ingress(), env.ID).Delete().Request(ctx, &hydrav1.DeleteEnvironmentRequest{})
			return err
		case "app":
			_, err := hydrav1.NewAppServiceIngressClient(tEnv.Ingress(), app.ID).Delete().Request(ctx, &hydrav1.DeleteAppRequest{})
			return err
		default:
			_, err := hydrav1.NewProjectServiceIngressClient(tEnv.Ingress(), project.ID).Delete().Request(ctx, &hydrav1.DeleteProjectRequest{})
			return err
		}
	}
	completed := make(chan error, 1)
	go func() { completed <- deleteResource() }()
	require.Eventually(t, func() bool {
		return countRows(t, ctx, h.DB, "SELECT COUNT(*) FROM environments WHERE id = ? AND deleting_at IS NOT NULL", env.ID) == 1
	}, 30*time.Second, 25*time.Millisecond)

	require.NoError(t, h.DB.DeleteDeploymentInstances(ctx, db.DeleteDeploymentInstancesParams{
		DeploymentID: deployment.ID, RegionID: region.ID,
	}))
	require.Equal(t, 0, countRows(t, ctx, h.DB, "SELECT COUNT(*) FROM instances WHERE deployment_id = ?", deployment.ID))
	for _, regionID := range []string{region.ID, secondRegion.ID} {
		select {
		case err := <-completed:
			t.Fatalf("deletion completed before every region confirmed removal: %v", err)
		case <-time.After(1500 * time.Millisecond):
		}
		for _, check := range checks[:4] {
			require.Equal(t, 1, countRows(t, ctx, h.DB, check.query, check.arg))
		}
		removed, err := h.DB.ConfirmDeploymentTopologyRemoval(ctx, db.ConfirmDeploymentTopologyRemovalParams{
			DeploymentID: deployment.ID, RegionID: regionID,
		})
		require.NoError(t, err)
		require.EqualValues(t, 1, removed)
	}
	select {
	case err := <-completed:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	require.NoError(t, deleteResource())
	runResourceCleanup(t, ctx, tEnv)

	for _, c := range checks {
		want := 0
		if c.survives {
			want = 1
		}
		require.Equal(t, want, countRows(t, ctx, h.DB, c.query, c.arg), c.query)
	}
	if appSurvives {
		app, err := h.DB.FindAppById(ctx, app.ID)
		require.NoError(t, err)
		require.False(t, app.CurrentDeploymentID.Valid)
	}
	siblingApp, err := h.DB.FindAppById(ctx, sibling.AppID)
	require.NoError(t, err)
	require.Equal(t, sql.NullString{String: sibling.ID, Valid: true}, siblingApp.CurrentDeploymentID)
	require.Positive(t, countRows(t, ctx, h.DB, "SELECT COUNT(*) FROM clickhouse_outbox WHERE workspace_id = ?", workspaceID))

	_, err = h.DB.RW().ExecContext(ctx, "DELETE FROM deployment_topology WHERE deployment_id = ?", sibling.ID)
	require.NoError(t, err)
	for _, id := range []string{project.ID, sibling.ProjectID} {
		_, err := hydrav1.NewProjectServiceIngressClient(tEnv.Ingress(), id).Delete().Request(ctx, &hydrav1.DeleteProjectRequest{})
		require.NoError(t, err)
	}
}

func TestResourceCleanupRepairsMissingParents(t *testing.T) {
	for _, parent := range []string{"environment", "app", "project"} {
		t.Run(parent, func(t *testing.T) { testResourceCleanupRepairsMissingParent(t, parent) })
	}
}

func testResourceCleanupRepairsMissingParent(t *testing.T, parent string) {
	t.Helper()
	h := New(t)
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	tEnv := startDeletionRestate(t, h)

	workspaceID := h.Seed.Resources.UserWorkspace.ID
	now := time.Now().UnixMilli()
	deployment := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: uid.DNS1035()})
	sibling := h.CreateDeployment(ctx, CreateDeploymentRequest{Region: uid.DNS1035()})
	var firstRegionID string
	err := h.DB.RO().QueryRowContext(ctx, "SELECT region_id FROM deployment_topology WHERE deployment_id = ?", deployment.ID).Scan(&firstRegionID)
	require.NoError(t, err)
	regionIDs := []string{firstRegionID, h.Seed.CreateRegion(ctx, seed.CreateRegionRequest{Name: uid.DNS1035(), Platform: "test"}).ID}
	require.NoError(t, h.DB.InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
		WorkspaceID: workspaceID, DeploymentID: deployment.ID, RegionID: regionIDs[1],
		AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
		DesiredStatus: db.DeploymentTopologyDesiredStatusRunning, CreatedAt: now,
	}))
	h.Seed.CreateInstance(ctx, seed.CreateInstanceRequest{
		WorkspaceID: workspaceID, ProjectID: deployment.ProjectID, AppID: deployment.AppID,
		DeploymentID: deployment.ID, RegionID: regionIDs[0], Address: "10.0.0.1",
	})
	require.NoError(t, h.DB.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID: uid.New("fr"), ProjectID: deployment.ProjectID, AppID: deployment.AppID, DeploymentID: deployment.ID,
		EnvironmentID: deployment.EnvironmentID, FullyQualifiedDomainName: uid.DNS1035() + ".example.com",
		Sticky: db.FrontlineRoutesStickyNone, CreatedAt: now,
	}))
	require.NoError(t, h.DB.InsertCiliumNetworkPolicy(ctx, db.InsertCiliumNetworkPolicyParams{
		ID: uid.New("cnp"), WorkspaceID: workspaceID, ProjectID: deployment.ProjectID, AppID: deployment.AppID,
		EnvironmentID: deployment.EnvironmentID, DeploymentID: deployment.ID, K8sName: uid.New("k8s"),
		K8sNamespace: "test-ns", RegionID: regionIDs[0], Policy: []byte("{}"), CreatedAt: now,
	}))
	require.NoError(t, h.DB.InsertAppEnvironmentVariable(ctx, db.InsertAppEnvironmentVariableParams{
		ID: uid.New("aev"), WorkspaceID: workspaceID, AppID: deployment.AppID, EnvironmentID: deployment.EnvironmentID,
		EnvKey: "TEST_KEY", Value: "test_value", CreatedAt: now,
	}))
	require.NoError(t, h.DB.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID: workspaceID, ProjectID: deployment.ProjectID, AppID: deployment.AppID,
		InstallationID: 12345, RepositoryID: 67890, RepositoryFullName: "unkeyed/test-repo", CreatedAt: now,
	}))
	portalID := uid.New(uid.PortalPrefix)
	_, err = h.DB.RW().ExecContext(ctx, `INSERT INTO portals (id, workspace_id, project_id, app_id, slug, display_name, created_at)
		VALUES (?, ?, ?, ?, ?, 'test', 1)`, portalID, workspaceID, deployment.ProjectID, deployment.AppID, portalID)
	require.NoError(t, err)

	missingEnvironment, missingApp, missingProject := parent == "environment", parent == "app", parent == "project"
	checks := []struct {
		query   string
		arg     string
		pending bool
		final   bool
	}{
		{"SELECT COUNT(*) FROM deployments WHERE id = ?", deployment.ID, true, false},
		{"SELECT COUNT(*) FROM instances WHERE deployment_id = ?", deployment.ID, true, false},
		{"SELECT COUNT(*) FROM frontline_routes WHERE deployment_id = ?", deployment.ID, true, false},
		{"SELECT COUNT(*) FROM cilium_network_policies WHERE deployment_id = ?", deployment.ID, true, false},
		{"SELECT COUNT(*) FROM environments WHERE id = ?", deployment.EnvironmentID, !missingEnvironment, false},
		{"SELECT COUNT(*) FROM app_environment_variables WHERE environment_id = ?", deployment.EnvironmentID, !missingEnvironment, false},
		{"SELECT COUNT(*) FROM apps WHERE id = ?", deployment.AppID, !missingApp, missingEnvironment},
		{"SELECT COUNT(*) FROM github_repo_connections WHERE app_id = ?", deployment.AppID, !missingApp, missingEnvironment},
		{"SELECT COUNT(*) FROM projects WHERE id = ?", deployment.ProjectID, !missingProject, !missingProject},
		{"SELECT COUNT(*) FROM portals WHERE id = ?", portalID, missingEnvironment, missingEnvironment},
		{"SELECT COUNT(*) FROM deployment_topology WHERE deployment_id = ? AND desired_status = 'running'", sibling.ID, true, true},
		{"SELECT COUNT(*) FROM environments WHERE id = ?", sibling.EnvironmentID, true, true},
	}
	requireRows := func(final bool) {
		t.Helper()
		for _, c := range checks {
			want := c.pending
			if final {
				want = c.final
			}
			wantCount := 0
			if want {
				wantCount = 1
			}
			require.Equal(t, wantCount, countRows(t, ctx, h.DB, c.query, c.arg), "%s [%s]", c.query, c.arg)
		}
	}

	switch parent {
	case "environment":
		_, err = h.DB.RW().ExecContext(ctx, "DELETE FROM environments WHERE id = ?", deployment.EnvironmentID)
	case "app":
		_, err = h.DB.RW().ExecContext(ctx, "DELETE FROM apps WHERE id = ?", deployment.AppID)
	default:
		_, err = h.DB.RW().ExecContext(ctx, "DELETE FROM projects WHERE id = ?", deployment.ProjectID)
	}
	require.NoError(t, err)

	runResourceCleanup(t, ctx, tEnv)
	require.Eventually(t, func() bool {
		return countRows(t, ctx, h.DB, "SELECT COUNT(*) FROM deployment_topology WHERE deployment_id = ? AND desired_status = 'stopped'", deployment.ID) == 2
	}, 30*time.Second, 25*time.Millisecond)
	for range 2 {
		runResourceCleanup(t, ctx, tEnv)
		requireRows(false)
	}

	for i, regionID := range regionIDs {
		require.Equal(t, len(regionIDs)-i, countRows(t, ctx, h.DB, "SELECT COUNT(*) FROM deployment_topology WHERE deployment_id = ?", deployment.ID))
		runResourceCleanup(t, ctx, tEnv)
		requireRows(false)
		removed, err := h.DB.ConfirmDeploymentTopologyRemoval(ctx, db.ConfirmDeploymentTopologyRemovalParams{
			DeploymentID: deployment.ID, RegionID: regionID,
		})
		require.NoError(t, err)
		require.EqualValues(t, 1, removed)
	}
	require.Eventually(t, func() bool {
		return countRows(t, ctx, h.DB, "SELECT COUNT(*) FROM deployments WHERE environment_id = ?", deployment.EnvironmentID) == 0 &&
			countRows(t, ctx, h.DB, "SELECT COUNT(*) FROM environments WHERE id = ?", deployment.EnvironmentID) == 0
	}, 60*time.Second, 100*time.Millisecond)

	runResourceCleanup(t, ctx, tEnv)
	requireRows(true)
}

func countRows(t *testing.T, ctx context.Context, database db.Database, query string, args ...any) int {
	t.Helper()
	var count int
	err := database.RO().QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err)
	return count
}
