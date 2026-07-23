package cron_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// The cron handler keeps terminal deployments for 30 days before deleting
// them, so the test seeds updated_at offsets safely on either side of that
// boundary.
const deploymentCleanupRetention = 30 * 24 * time.Hour

func TestRunDeploymentCleanup_Integration(t *testing.T) {
	h := harness.New(t)

	now := time.Now()
	staleAt := now.Add(-(deploymentCleanupRetention + 10*24*time.Hour)).UnixMilli()
	recentAt := now.Add(-10 * 24 * time.Hour).UnixMilli()

	scope := seedDeploymentCleanupScope(t, h)

	seedDeployment := func(status mysqltype.DeploymentsStatus, updatedAt int64) db.Deployment {
		return h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			ID:            "",
			WorkspaceID:   scope.workspaceID,
			ProjectID:     scope.projectID,
			AppID:         scope.appID,
			EnvironmentID: scope.environmentID,
			Status:        status,
			CreatedAt:     updatedAt,
			UpdatedAt:     sql.NullInt64{Int64: updatedAt, Valid: true},
		})
	}

	staleStopped := seedDeployment(mysqltype.DeploymentsStatusStopped, staleAt)
	staleFailed := seedDeployment(mysqltype.DeploymentsStatusFailed, staleAt)
	seedDeploymentDependents(t, h, staleFailed, staleAt)
	staleCancelled := seedDeployment(mysqltype.DeploymentsStatusCancelled, staleAt)
	// recently stopped -> inside the retention window, must survive.
	recentStopped := seedDeployment(mysqltype.DeploymentsStatusStopped, recentAt)
	// stale but ready -> could still serve traffic, must survive.
	staleReady := seedDeployment(mysqltype.DeploymentsStatusReady, staleAt)
	// A stale failed deployment referenced as current must survive even though
	// the status would otherwise be prunable.
	staleCurrent := seedDeployment(mysqltype.DeploymentsStatusFailed, staleAt)
	err := h.DB.UpdateAppDeployments(h.Ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: staleCurrent.ID, Valid: true},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		AppID:               scope.appID,
	})
	require.NoError(t, err)

	client := hydrav1.NewCronServiceIngressClient(h.Restate, "deployment-cleanup")
	resp, err := client.RunDeploymentCleanup().Request(h.Ctx, &hydrav1.RunDeploymentCleanupRequest{})
	require.NoError(t, err)

	require.Equal(t, int64(2), resp.GetDeploymentsDeleted(), "only non-recoverable stale deployments are deleted")

	requireDeploymentExists(t, h, staleStopped.ID, true)
	requireDeploymentExists(t, h, staleFailed.ID, false)
	requireNoDeploymentDependents(t, h, staleFailed.ID)
	requireDeploymentExists(t, h, staleCancelled.ID, false)
	requireDeploymentExists(t, h, recentStopped.ID, true)
	requireDeploymentExists(t, h, staleReady.ID, true)
	requireDeploymentExists(t, h, staleCurrent.ID, true)
}

// TestRunDeploymentCleanup_RollsBackBatchOnDeleteFailure proves dependent
// deletes do not commit when the final deployment delete fails.
func TestRunDeploymentCleanup_RollsBackBatchOnDeleteFailure(t *testing.T) {
	h := harness.New(t)
	scope := seedDeploymentCleanupScope(t, h)
	staleAt := time.Now().Add(-(deploymentCleanupRetention + 24*time.Hour)).UnixMilli()
	deployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   scope.workspaceID,
		ProjectID:     scope.projectID,
		AppID:         scope.appID,
		EnvironmentID: scope.environmentID,
		Status:        mysqltype.DeploymentsStatusFailed,
		CreatedAt:     staleAt,
		UpdatedAt:     sql.NullInt64{Int64: staleAt, Valid: true},
	})
	seedDeploymentDependents(t, h, deployment, staleAt)

	triggerName := uid.New("cleanup_failure")
	_, err := h.DB.RW().ExecContext(h.Ctx, fmt.Sprintf(
		"CREATE TRIGGER `%s` BEFORE DELETE ON deployments FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected cleanup failure'",
		triggerName,
	))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, dropErr := h.DB.RW().ExecContext(h.Ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS `%s`", triggerName))
		require.NoError(t, dropErr)
	})

	client := hydrav1.NewCronServiceIngressClient(h.Restate, "deployment-cleanup")
	_, err = client.RunDeploymentCleanup().Request(h.Ctx, &hydrav1.RunDeploymentCleanupRequest{})
	require.Error(t, err)

	requireDeploymentExists(t, h, deployment.ID, true)
	for _, table := range deploymentDependentTables {
		require.Equal(t, int64(1), countDeploymentDependents(t, h, table, deployment.ID), "%s should roll back", table)
	}
}

type deploymentCleanupScope struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

// seedDeploymentCleanupScope creates the ownership tree required by deployments.
func seedDeploymentCleanupScope(t *testing.T, h *harness.Harness) deploymentCleanupScope {
	t.Helper()
	workspace := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	environment := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})
	return deploymentCleanupScope{
		workspaceID:   workspace.ID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

// deploymentDependentTables lists every table removed in the cleanup transaction.
var deploymentDependentTables = []string{
	"deployment_steps",
	"deployment_topology",
	"instances",
	"frontline_routes",
	"cilium_network_policies",
	"openapi_specs",
}

// seedDeploymentDependents creates one row in every table cleanup must remove.
func seedDeploymentDependents(t *testing.T, h *harness.Harness, deployment db.Deployment, createdAt int64) {
	t.Helper()
	regionID := uid.New(uid.RegionPrefix)
	statements := []string{
		`INSERT INTO deployment_steps (workspace_id, project_id, environment_id, deployment_id, app_id, step, started_at) VALUES (?, ?, ?, ?, ?, 'queued', ?)`,
		`INSERT INTO deployment_topology (workspace_id, deployment_id, region_id, desired_status, created_at) VALUES (?, ?, ?, 'stopped', ?)`,
		`INSERT INTO instances (id, deployment_id, workspace_id, project_id, app_id, region_id, k8s_name, address, cpu_millicores, memory_mib, status) VALUES (?, ?, ?, ?, ?, ?, ?, '127.0.0.1', 100, 128, 'inactive')`,
		`INSERT INTO frontline_routes (id, project_id, app_id, deployment_id, environment_id, fully_qualified_domain_name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		`INSERT INTO cilium_network_policies (id, workspace_id, project_id, app_id, environment_id, deployment_id, k8s_name, k8s_namespace, region_id, policy, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'default', ?, '{}', ?)`,
		`INSERT INTO openapi_specs (id, workspace_id, deployment_id, content, created_at) VALUES (?, ?, ?, 'test', ?)`,
	}
	args := [][]any{
		{deployment.WorkspaceID, deployment.ProjectID, deployment.EnvironmentID, deployment.ID, deployment.AppID, createdAt},
		{deployment.WorkspaceID, deployment.ID, regionID, createdAt},
		{uid.New(uid.InstancePrefix), deployment.ID, deployment.WorkspaceID, deployment.ProjectID, deployment.AppID, regionID, uid.New("instance")},
		{uid.New(uid.FrontlineRoutePrefix), deployment.ProjectID, deployment.AppID, deployment.ID, deployment.EnvironmentID, uid.New("cleanup") + ".test", createdAt},
		{uid.New(uid.CiliumNetworkPolicyPrefix), deployment.WorkspaceID, deployment.ProjectID, deployment.AppID, deployment.EnvironmentID, deployment.ID, uid.New("policy"), regionID, createdAt},
		{uid.New(uid.OpenApiSpecPrefix), deployment.WorkspaceID, deployment.ID, createdAt},
	}
	for i, statement := range statements {
		_, err := h.DB.RW().ExecContext(h.Ctx, statement, args[i]...)
		require.NoError(t, err)
	}
}

// requireNoDeploymentDependents verifies the transactional cleanup contract.
func requireNoDeploymentDependents(t *testing.T, h *harness.Harness, deploymentID string) {
	t.Helper()
	for _, table := range deploymentDependentTables {
		require.Zero(t, countDeploymentDependents(t, h, table, deploymentID), "%s still references deployment %s", table, deploymentID)
	}
}

// countDeploymentDependents returns references from one controlled table name.
func countDeploymentDependents(t *testing.T, h *harness.Harness, table, deploymentID string) int64 {
	t.Helper()
	var count int64
	err := h.DB.RO().QueryRowContext(h.Ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deployment_id = ?", table), deploymentID).Scan(&count)
	require.NoError(t, err)
	return count
}

func requireDeploymentExists(t *testing.T, h *harness.Harness, id string, want bool) {
	t.Helper()
	_, err := h.DB.FindDeploymentById(h.Ctx, id)
	if want {
		require.NoError(t, err, "deployment %s should still exist", id)
		return
	}
	require.True(t, db.IsNotFound(err), "deployment %s should be deleted, got err=%v", id, err)
}
