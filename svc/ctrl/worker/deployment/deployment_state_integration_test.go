package deployment_test

import (
	"testing"
	"time"

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
		Status:        db.DeploymentsStatusReady,
	})

	client := hydrav1.NewDeploymentServiceIngressClient(h.Restate, dep.ID)
	_, err := client.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		DelayMillis: 500,
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:   true,
	})
	require.NoError(t, err)

	_, err = h.DB.RW().ExecContext(h.Ctx, "DELETE FROM deployment_topology WHERE deployment_id = ?", dep.ID)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(h.Ctx, "DELETE FROM deployments WHERE id = ?", dep.ID)
	require.NoError(t, err)

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
