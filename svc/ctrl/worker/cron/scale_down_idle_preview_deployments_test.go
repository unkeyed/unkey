package cron_test

import (
	"database/sql"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestScaleDownIdlePreviewDeployments_ScalesDownIdleDeploymentWithZeroRequests
// guarantees that the cron workflow can use persisted deployment state and
// request telemetry together to stop an idle preview deployment.
func TestScaleDownIdlePreviewDeployments_ScalesDownIdleDeploymentWithZeroRequests(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	triggerScaleDown(t, h)

	requireDeploymentStopped(t, h, dep.ID)
}

// TestScaleDownIdlePreviewDeployments_DoesNotScaleDownDeploymentWithRecentRequests
// guarantees that recent traffic in ClickHouse keeps a preview deployment
// desired to run even when its database timestamps are old.
func TestScaleDownIdlePreviewDeployments_DoesNotScaleDownDeploymentWithRecentRequests(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	h.ClickHouseSeed.InsertFrontlineRequests(h.Ctx, ws.ID, project.ID, app.ID, env.ID, dep.ID, 10, time.Now().Add(-1*time.Hour))

	triggerScaleDown(t, h)

	requireDeploymentStillRunning(t, h, dep.ID)
}

// TestScaleDownIdlePreviewDeployments_IgnoresNonPreviewEnvironments guarantees
// that the cron workflow only reclaims preview deployments.
func TestScaleDownIdlePreviewDeployments_IgnoresNonPreviewEnvironments(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Kind:             mysqltype.EnvironmentKindProduction,
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	triggerScaleDown(t, h)

	requireDeploymentStillRunning(t, h, dep.ID)
}

// TestScaleDownIdlePreviewDeployments_IgnoresDeploymentsNotInReadyStatus
// guarantees that the cron workflow does not stop deployments still moving
// through another lifecycle transition.
func TestScaleDownIdlePreviewDeployments_IgnoresDeploymentsNotInReadyStatus(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusPending,
		CreatedAt:     oldTime,
		UpdatedAt:     oldUpdatedAt,
	})

	triggerScaleDown(t, h)

	requireDeploymentStillRunning(t, h, dep.ID)
}

// TestScaleDownIdlePreviewDeployments_IgnoresRecentlyCreatedDeployments
// guarantees that a new preview deployment gets the full idle window before
// cron can stop it.
func TestScaleDownIdlePreviewDeployments_IgnoresRecentlyCreatedDeployments(t *testing.T) {
	h := harness.New(t)

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	recentTime := time.Now().Add(-5 * time.Minute).UnixMilli()
	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     recentTime,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: recentTime},
	})

	triggerScaleDown(t, h)

	requireDeploymentStillRunning(t, h, dep.ID)
}

// TestScaleDownIdlePreviewDeployments_IgnoresRecentlyUpdatedDeployments
// guarantees that recent deployment changes reset the idle window.
func TestScaleDownIdlePreviewDeployments_IgnoresRecentlyUpdatedDeployments(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	recentUpdate := time.Now().Add(-5 * time.Minute).UnixMilli()
	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
		CreatedAt:     oldTime,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: recentUpdate},
	})

	triggerScaleDown(t, h)

	requireDeploymentStillRunning(t, h, dep.ID)
}

// TestScaleDownIdlePreviewDeployments_HandlesMultipleDeploymentsAcrossMultipleEnvironments
// guarantees that one cron pass evaluates each preview deployment independently
// across environments and workspaces.
func TestScaleDownIdlePreviewDeployments_HandlesMultipleDeploymentsAcrossMultipleEnvironments(t *testing.T) {
	h := harness.New(t)
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	var idleDeployments []db.Deployment
	var activeDeployments []db.Deployment

	for range 2 {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      ws.ID,
			Name:             "test-project",
			Slug:             uid.New("slug"),
			DeleteProtection: false,
		})
		app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: ws.ID,
			ProjectID:   project.ID,
			Name:        "default",
			Slug:        "default",
		})
		env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
			ID:               uid.New(uid.EnvironmentPrefix),
			WorkspaceID:      ws.ID,
			ProjectID:        project.ID,
			AppID:            app.ID,
			Slug:             "preview",
			Description:      "",
			SentinelConfig:   nil,
			DeleteProtection: false,
		})

		idle := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			ID:            "",
			WorkspaceID:   ws.ID,
			ProjectID:     project.ID,
			AppID:         app.ID,
			EnvironmentID: env.ID,
			Status:        mysqltype.DeploymentsStatusReady,
			CreatedAt:     oldTime,
			UpdatedAt:     oldUpdatedAt,
		})
		idleDeployments = append(idleDeployments, idle)

		active := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			ID:            "",
			WorkspaceID:   ws.ID,
			ProjectID:     project.ID,
			AppID:         app.ID,
			EnvironmentID: env.ID,
			Status:        mysqltype.DeploymentsStatusReady,
			CreatedAt:     oldTime,
			UpdatedAt:     oldUpdatedAt,
		})
		h.ClickHouseSeed.InsertFrontlineRequests(h.Ctx, ws.ID, project.ID, app.ID, env.ID, active.ID, 5, time.Now().Add(-30*time.Minute))
		activeDeployments = append(activeDeployments, active)
	}

	triggerScaleDown(t, h)

	for _, dep := range idleDeployments {
		requireDeploymentStopped(t, h, dep.ID)
	}
	for _, dep := range activeDeployments {
		requireDeploymentStillRunning(t, h, dep.ID)
	}
}

// TestScaleDownIdlePreviewDeployments_PaginatesAcrossManyPreviewEnvironmentsAtScale
// guarantees that the cron workflow does not stop after the first preview
// environment page.
func TestScaleDownIdlePreviewDeployments_PaginatesAcrossManyPreviewEnvironmentsAtScale(t *testing.T) {
	h := harness.New(t, harness.WithTimeout(5*time.Minute))
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()
	oldUpdatedAt := sql.NullInt64{Valid: true, Int64: oldTime}

	const count = 201
	depIDs := make([]string, count)
	for i := range count {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      ws.ID,
			Name:             "test-project",
			Slug:             uid.New("slug"),
			DeleteProtection: false,
		})
		app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: ws.ID,
			ProjectID:   project.ID,
			Name:        "default",
			Slug:        "default",
		})
		env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
			ID:               uid.New(uid.EnvironmentPrefix),
			WorkspaceID:      ws.ID,
			ProjectID:        project.ID,
			AppID:            app.ID,
			Slug:             "preview",
			Description:      "",
			SentinelConfig:   nil,
			DeleteProtection: false,
		})

		dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			ID:            "",
			WorkspaceID:   project.WorkspaceID,
			ProjectID:     project.ID,
			AppID:         app.ID,
			EnvironmentID: env.ID,
			Status:        mysqltype.DeploymentsStatusReady,
			CreatedAt:     oldTime,
			UpdatedAt:     oldUpdatedAt,
		})
		depIDs[i] = dep.ID
	}

	triggerScaleDown(t, h)

	for _, id := range depIDs {
		requireDeploymentStopped(t, h, id)
	}
}

// triggerScaleDown runs the cron workflow through Restate ingress so tests
// cover the same serialized handler path as production.
func triggerScaleDown(t *testing.T, h *harness.Harness) {
	t.Helper()

	client := hydrav1.NewCronServiceIngressClient(h.Restate, "idle-preview-deployments")
	_, err := client.RunScaleDownIdlePreviewDeployments().Request(h.Ctx, &hydrav1.RunScaleDownIdlePreviewDeploymentsRequest{})
	require.NoError(t, err)
}

// requireDeploymentStopped waits for a scheduled scale-down to reach MySQL.
//
// The cron handler never writes desired_state itself. It calls
// DeploymentService.ScheduleDesiredStateChange, which records the transition on
// the deployment's virtual object and then Sends ChangeDesiredState, and only
// that send updates the row. The send is asynchronous, so it can still be
// queued when triggerScaleDown returns, and the wait grows with the number of
// deployments the scan scheduled ahead of this one.
func requireDeploymentStopped(t *testing.T, h *harness.Harness, deploymentID string) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		updated, err := h.DB.FindDeploymentById(h.Ctx, deploymentID)
		if !assert.NoError(c, err) {
			return
		}
		assert.Equal(c, mysqltype.DeploymentsDesiredStateStopped, updated.DesiredState,
			"deployment %s should be stopped", deploymentID)
	}, 60*time.Second, 100*time.Millisecond)
}

// requireDeploymentStillRunning asserts that the scan left a deployment alone.
//
// Reading MySQL straight after triggerScaleDown can observe "running" merely
// because a wrongly scheduled stop has not been applied yet, which would let a
// real regression pass. Requesting something on the same virtual object first
// closes that window: the object runs one invocation at a time for a given key,
// and any ChangeDesiredState for this deployment was enqueued during the scan,
// so it has been processed by the time this request returns. Clearing a
// transition that should not exist in the first place is a noop.
func requireDeploymentStillRunning(t *testing.T, h *harness.Harness, deploymentID string) {
	t.Helper()

	_, err := hydrav1.NewDeploymentServiceIngressClient(h.Restate, deploymentID).
		ClearScheduledStateChanges().
		Request(h.Ctx, &hydrav1.ClearScheduledStateChangesRequest{})
	require.NoError(t, err)

	updated, err := h.DB.FindDeploymentById(h.Ctx, deploymentID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateRunning, updated.DesiredState,
		"deployment %s should still be running", deploymentID)
}
