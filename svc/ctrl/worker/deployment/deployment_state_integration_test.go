package deployment_test

import (
	"database/sql"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
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
	capture := loggertest.Install(t)

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
		return h.Seed.CreateAppConnection(h.Ctx, seed.CreateAppConnectionRequest{
			WorkspaceID:         ws.ID,
			ProjectID:           project.ID,
			CallerAppID:         app.ID,
			CallerEnvironmentID: env.ID,
			TargetAppID:         app.ID,
			Name:                uid.New("name"),
			ResourceType:        resourceType,
			SelectionMode:       db.ConnectionAppTargetsSelectionModeDeployment,
			TargetDeploymentID:  sql.NullString{String: deploymentID, Valid: true},
		})
	}

	waitForDesiredState := func(deploymentID string, state mysqltype.DeploymentsDesiredState) {
		t.Helper()
		require.Eventually(t, func() bool {
			deployment, err := h.DB.FindDeploymentById(h.Ctx, deploymentID)
			return err == nil && deployment.DesiredState == state
		}, 90*time.Second, 100*time.Millisecond)
	}

	automatic := createDeployment()
	connectionID := bind(automatic.ID, "app")
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
	}, 500*time.Millisecond, 50*time.Millisecond, "a pinned deployment stopped while its connection existed")
	require.Eventually(t, func() bool {
		for _, record := range capture.Records() {
			if record.Message == "deployment stop deferred because an app connection pins it" &&
				loggertest.FlatAttrs(record)["deployment_id"] == automatic.ID {
				return true
			}
		}
		return false
	}, 5*time.Second, 50*time.Millisecond, "a deferred stop must name the pinned deployment in the logs")

	callerApp := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: ws.ID, ProjectID: project.ID,
		Name: "caller", Slug: "caller",
	})
	callerEnv := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: ws.ID, ProjectID: project.ID,
		AppID: callerApp.ID, Slug: "preview",
	})
	caller := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID: ws.ID, ProjectID: project.ID, AppID: callerApp.ID, EnvironmentID: callerEnv.ID,
		Status: mysqltype.DeploymentsStatusReady, Capabilities: mysqltype.DeploymentCapabilities{PrivateNetworking: true},
	})
	require.NoError(t, h.DB.InsertDeploymentConnection(h.Ctx, db.InsertDeploymentConnectionParams{
		DeploymentID: caller.ID, ConnectionID: connectionID, WorkspaceID: ws.ID, ProjectID: project.ID,
		AppID: callerApp.ID, EnvironmentID: callerEnv.ID, ResourceType: "app", ResourceID: app.ID,
		Name: "pinned", CreatedAt: time.Now().UnixMilli(),
	}))
	require.NoError(t, h.DB.InsertDeploymentConnectionAppTarget(h.Ctx, db.InsertDeploymentConnectionAppTargetParams{
		DeploymentID: caller.ID, ConnectionID: connectionID,
		SelectionMode:      db.DeploymentConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{String: automatic.ID, Valid: true},
	}))
	require.NoError(t, h.DB.DeleteConnectionAppTargetByConnectionId(h.Ctx, connectionID))
	deleted, err := h.DB.DeleteAppConnectionById(h.Ctx, connectionID)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	recordsBefore := len(capture.Records())
	_, err = automaticClient.ScheduleDesiredStateChange().Request(h.Ctx, &hydrav1.ScheduleDesiredStateChangeRequest{
		State:            hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		Overwrite:        true,
		DeferWhilePinned: true,
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		for _, record := range capture.Records()[recordsBefore:] {
			if record.Message == "deployment stop deferred because an app connection pins it" &&
				loggertest.FlatAttrs(record)["deployment_id"] == automatic.ID {
				return true
			}
		}
		return false
	}, 5*time.Second, 50*time.Millisecond)
	stillPinned, err := h.DB.FindDeploymentById(h.Ctx, automatic.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsDesiredStateRunning, stillPinned.DesiredState)
	require.NoError(t, h.DB.UpdateDeploymentDesiredState(h.Ctx, db.UpdateDeploymentDesiredStateParams{
		DesiredState: mysqltype.DeploymentsDesiredStateStopped,
		UpdatedAt:    sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:           caller.ID,
	}))
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
