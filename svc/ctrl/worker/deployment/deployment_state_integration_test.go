package deployment_test

import (
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestChangeDesiredState_NoOpsWhenDeploymentDeleted verifies that a delayed
// ChangeDesiredState call succeeds when the deployment row was removed after
// scheduling, for example by an environment delete cascade.
func TestChangeDesiredState_NoOpsWhenDeploymentDeleted(t *testing.T) {
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
	dep := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	client := hydrav1.NewDeploymentServiceIngressClient(h.Restate, dep.ID)
	_, err := client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 500,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)

	require.NoError(t, h.DB.DeleteDeploymentTopologiesByEnvironmentId(h.Ctx, env.ID))
	require.NoError(t, h.DB.DeleteDeploymentsByEnvironmentId(h.Ctx, env.ID))

	time.Sleep(time.Second)

	_, err = h.DB.FindDeploymentById(h.Ctx, dep.ID)
	require.Error(t, err)
	require.True(t, db.IsNotFound(err))

	_, err = client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 0,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)
}

func TestChangeDesiredState_PinnedDeploymentLifecycle(t *testing.T) {
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
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
	})

	createDeployment := func() db.Deployment {
		t.Helper()
		return h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			WorkspaceID:   ws.ID,
			ProjectID:     project.ID,
			AppID:         app.ID,
			EnvironmentID: env.ID,
			Status:        mysqltype.DeploymentsStatusReady,
		})
	}

	bind := func(deploymentID, resourceType string) string {
		t.Helper()
		bindingID := uid.New("binding")
		_, err := h.DB.RW().ExecContext(h.Ctx, `INSERT INTO app_bindings
			(id, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, name, selection_mode, target_deployment_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'deployment', ?, ?)`,
			bindingID, ws.ID, project.ID, app.ID, env.ID, resourceType, app.ID, uid.New("name"), deploymentID, time.Now().UnixMilli())
		require.NoError(t, err)
		return bindingID
	}

	waitForDesiredState := func(deploymentID string, state mysqltype.DeploymentsDesiredState) {
		t.Helper()
		require.Eventually(t, func() bool {
			deployment, err := h.DB.FindDeploymentById(h.Ctx, deploymentID)
			return err == nil && deployment.DesiredState == state
		}, 90*time.Second, 100*time.Millisecond)
	}

	automatic := createDeployment()
	bindingID := bind(automatic.ID, "app")
	automaticClient := hydrav1.NewDeploymentServiceIngressClient(h.Restate, automatic.ID)
	_, err := automaticClient.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis:      0,
		State:            hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:        true,
		DeferWhilePinned: true,
	})
	require.NoError(t, err)
	require.Never(t, func() bool {
		deployment, err := h.DB.FindDeploymentById(h.Ctx, automatic.ID)
		return err != nil || deployment.DesiredState != mysqltype.DeploymentsDesiredStateRunning
	}, 500*time.Millisecond, 50*time.Millisecond, "a pinned deployment stopped while its binding existed")

	result, err := h.DB.RW().ExecContext(h.Ctx, "DELETE FROM app_bindings WHERE id = ?", bindingID)
	require.NoError(t, err)
	deleted, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	waitForDesiredState(automatic.ID, mysqltype.DeploymentsDesiredStateStopped)

	explicit := createDeployment()
	bind(explicit.ID, "app")
	explicitClient := hydrav1.NewDeploymentServiceIngressClient(h.Restate, explicit.ID)
	_, err = explicitClient.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		State:     hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite: true,
	})
	require.NoError(t, err)
	waitForDesiredState(explicit.ID, mysqltype.DeploymentsDesiredStateStopped)

	otherResource := createDeployment()
	bind(otherResource.ID, "queue")
	otherResourceClient := hydrav1.NewDeploymentServiceIngressClient(h.Restate, otherResource.ID)
	_, err = otherResourceClient.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		State:            hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:        true,
		DeferWhilePinned: true,
	})
	require.NoError(t, err)
	waitForDesiredState(otherResource.ID, mysqltype.DeploymentsDesiredStateStopped)
}
